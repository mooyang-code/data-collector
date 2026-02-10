"""
data-collector Python 插件

HTTP 服务（端口 9001），由 scf-framework 的 HTTPPluginAdapter 调用。

接口：
  GET  /health       - 健康检查
  POST /on-trigger   - 接收触发事件（scheduled-collect）

触发事件 payload 中包含 TaskStore 快照（tasks + tasks_md5），
插件根据任务实例执行对应的采集逻辑。
DNS 解析由框架层（trpc.dns.timer）驱动，结果通过 metadata["dns_records"] 注入。
采集结果以 DataPoint 格式返回，由 scf-framework 统一写入 xData。

插件自动发现机制：启动时扫描 plugin 目录下所有 *.py 文件，
自动注册含 COLLECTOR 变量的模块。新增插件无需修改 main.py。
"""

import glob
import importlib
import json
import logging
import os
import sys
from typing import Optional
from http.server import HTTPServer, BaseHTTPRequestHandler

# ============================================================================
# 路径 & 日志初始化
# ============================================================================

_PLUGIN_DIR = os.path.abspath(os.path.dirname(__file__))
sys.path.insert(0, _PLUGIN_DIR)

_FRAMEWORK_PYTHON_DIR = os.path.join(_PLUGIN_DIR, "..", "..", "scf-framework", "python")
if os.path.isdir(_FRAMEWORK_PYTHON_DIR):
    sys.path.insert(0, os.path.abspath(_FRAMEWORK_PYTHON_DIR))

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger("data-collector-plugin")

# ============================================================================
# 模块级状态
# ============================================================================

_cls_handler = None
_log_context_initialized = False
dns_records: dict[str, dict] = {}
_collector_registry: dict[str, dict] = {}

# ============================================================================
# 入口
# ============================================================================

def main():
    global _cls_handler

    port = 9001
    logger.info(f"data-collector 插件启动，监听端口: {port}")

    # 初始化 CLS 日志上报
    config_path = os.path.join(_PLUGIN_DIR, "..", "configs", "config.yaml")
    try:
        from scf_log import setup_cls_logging
        _cls_handler = setup_cls_logging(
            config_path=config_path,
            level=logging.INFO,
            logger_name="data-collector-plugin",
            cls_key="cls",
        )
        logger.info("CLS 日志上报已启用")
    except Exception as e:
        logger.warning(f"CLS 日志上报初始化失败（降级为仅控制台日志）: {e}")

    # 自动发现并注册采集插件
    _discover_collectors()

    server = HTTPServer(("0.0.0.0", port), PluginHandler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info("插件服务停止")
        server.shutdown()
    finally:
        if _cls_handler is not None:
            logger.info("关闭 CLS 日志 handler，flush 剩余日志...")
            _cls_handler.close()


# ============================================================================
# HTTP Handler
# ============================================================================

class PluginHandler(BaseHTTPRequestHandler):
    """插件 HTTP 请求处理器"""

    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"status": "ok"}).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path == "/on-trigger":
            content_length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(content_length)
            try:
                event = json.loads(body)
                result = handle_trigger(event)
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps(result).encode())
            except BrokenPipeError:
                logger.warning("处理完成，但调用方已断开连接（超时）")
            except Exception as e:
                logger.error(f"处理触发事件失败: {e}", exc_info=True)
                try:
                    self.send_response(500)
                    self.send_header("Content-Type", "application/json")
                    self.end_headers()
                    self.wfile.write(json.dumps({"error": str(e)}).encode())
                except BrokenPipeError:
                    pass
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format, *args):
        logger.debug(f"{self.client_address[0]} - {format % args}")


# ============================================================================
# 触发事件处理
# ============================================================================

def handle_trigger(event: dict) -> dict:
    """处理 scheduled-collect 触发事件，分发到对应插件并返回结果。"""
    trigger_name = event.get("name", "")
    metadata = event.get("metadata") or {}

    _update_log_context(metadata)
    _update_dns_records(metadata)

    payload = _parse_payload(event.get("payload"))

    logger.info(f"收到触发事件: name={trigger_name}, "
                f"jobs_count={len(payload.get('jobs', []))}")

    if trigger_name == "scheduled-collect":
        return _dispatch_jobs(payload)

    logger.warning(f"未知触发器: {trigger_name}")
    return {"status": "ok"}


# ============================================================================
# 调度逻辑
# ============================================================================

def _dispatch_jobs(payload: dict) -> dict:
    """按 data_type 分发 jobs 到对应插件，收集结果。"""
    jobs_data = payload.get("jobs", [])
    logger.info(f"[_dispatch_jobs] 已注册的 collector: {list(_collector_registry.keys())}, "
                f"payload keys: {list(payload.keys())}, jobs_count: {len(jobs_data)}")
    if not jobs_data:
        logger.warning("[_dispatch_jobs] jobs 为空，直接返回")
        return {"status": "ok"}

    # 按 data_type 分组
    grouped: dict[str, list[dict]] = {}
    for i, job_raw in enumerate(jobs_data):
        task = job_raw.get("task", {})
        task_params_raw = task.get("task_params", "")
        if not task:
            logger.warning(f"[_dispatch_jobs] job[{i}] 缺少 task 字段, keys={list(job_raw.keys())}")
            continue
        try:
            params = json.loads(task_params_raw) if task_params_raw else {}
        except (json.JSONDecodeError, TypeError) as e:
            logger.warning(f"[_dispatch_jobs] job[{i}] task_params 解析失败: {e}, raw={task_params_raw[:200]}")
            continue
        data_type = params.get("data_type", "")
        if data_type not in _collector_registry:
            logger.warning(f"[_dispatch_jobs] job[{i}] 跳过未注册的 data_type={data_type}, "
                           f"已注册={list(_collector_registry.keys())}")
            continue
        grouped.setdefault(data_type, []).append(job_raw)

    all_task_results = []
    write_groups = []

    logger.info(f"[_dispatch_jobs] 分组结果: { {k: len(v) for k, v in grouped.items()} }")

    for data_type, raw_jobs in grouped.items():
        collector = _collector_registry[data_type]
        parse_fn = collector.get("parse_job")
        collect_fn = collector["collect"]

        parsed_jobs = []
        for raw in raw_jobs:
            parsed = parse_fn(raw) if parse_fn else raw
            if parsed is not None:
                parsed_jobs.append(parsed)
            else:
                logger.warning(f"[_dispatch_jobs] parse_job 返回 None, data_type={data_type}, "
                               f"task_id={raw.get('task', {}).get('task_id', 'N/A')}")

        if not parsed_jobs:
            logger.warning(f"[_dispatch_jobs] data_type={data_type} 所有 job 解析后为空，跳过采集")
            continue

        result = collect_fn(parsed_jobs, get_best_ip)
        logger.info(f"[_dispatch_jobs] data_type={data_type} 采集结果: "
                    f"task_results={len(result.get('task_results', []))}, "
                    f"write_groups={len(result.get('write_groups', []))}")
        all_task_results.extend(result.get("task_results", []))
        write_groups.extend(result.get("write_groups", []))

    response = {}
    if all_task_results:
        response["task_results"] = all_task_results
    if write_groups:
        response["write_groups"] = write_groups
    return response if response else {"status": "ok"}


# ============================================================================
# 插件自动发现
# ============================================================================

def _discover_collectors():
    """扫描 plugin 目录下所有 py 文件，自动注册含 COLLECTOR 变量的模块。"""
    scanned = []
    for filepath in glob.glob(os.path.join(_PLUGIN_DIR, "*.py")):
        filename = os.path.basename(filepath)
        if filename == "main.py" or filename.startswith("_"):
            continue
        module_name = filename[:-3]
        scanned.append(module_name)
        try:
            mod = importlib.import_module(module_name)
            collector = getattr(mod, "COLLECTOR", None)
            if collector and isinstance(collector, dict) and "data_type" in collector:
                data_type = collector["data_type"]
                _collector_registry[data_type] = collector
                logger.info(f"已注册采集插件: data_type={data_type}, module={module_name}")
            else:
                logger.info(f"模块 {module_name} 未定义 COLLECTOR，跳过")
        except Exception as e:
            logger.warning(f"加载插件模块 {module_name} 失败: {e}", exc_info=True)
    logger.info(f"插件扫描完成: 扫描模块={scanned}, 已注册={list(_collector_registry.keys())}")


# ============================================================================
# 内部工具函数
# ============================================================================

def _update_log_context(metadata: dict):
    """从触发事件的 metadata 中提取 nodeID/version，设置为 CLS 日志上下文字段。"""
    global _log_context_initialized
    if _cls_handler is None or _log_context_initialized:
        return
    node_id = metadata.get("nodeID", "")
    version = metadata.get("version", "")
    if node_id:
        _cls_handler.set_context_fields(nodeID=node_id, version=version)
        _log_context_initialized = True


def _update_dns_records(metadata: dict):
    """从触发事件的 metadata 中提取框架注入的 dns_records。"""
    global dns_records
    dns_json = metadata.get("dns_records", "")
    if not dns_json:
        return
    try:
        records = json.loads(dns_json)
        if records and isinstance(records, dict):
            dns_records = records
            domains = list(records.keys())
            logger.info(f"dns_records 已从 metadata 更新: domains={domains}")
    except (json.JSONDecodeError, TypeError) as e:
        logger.warning(f"解析 metadata dns_records 失败: {e}")


def get_best_ip(domain: str) -> Optional[str]:
    """获取域名的最优 IP（从框架注入的 dns_records 中读取）。"""
    record = dns_records.get(domain)
    if not record:
        return None
    for ip_info in record.get("ip_list", []):
        if ip_info.get("available"):
            return ip_info.get("ip")
    return None


def _parse_payload(payload_raw) -> dict:
    """将 event 中的 payload（str / dict / None）统一解析为 dict。"""
    if not payload_raw:
        return {}
    if isinstance(payload_raw, dict):
        return payload_raw
    if isinstance(payload_raw, str):
        try:
            return json.loads(payload_raw)
        except (json.JSONDecodeError, ValueError):
            logger.warning("payload json.loads failed")
    return {}


if __name__ == "__main__":
    main()
