# Data Collector - 数据采集云函数

## 一、项目概述

Data Collector 是一个基于 scf-framework 的数据采集云函数，运行在腾讯云 SCF Web 函数模式下。采用 **Go + Python 双进程架构**：Go 进程负责框架调度（心跳、触发器、DNS 代理、网关），Python 进程负责实际的数据采集逻辑。

Python 端采用**插件自动发现机制**：启动时扫描 `plugin/` 目录下所有 `*.py` 文件，自动注册含 `COLLECTOR` 字典的模块。新增采集插件无需修改 `main.py`。

当前已实现的插件：

| 插件文件 | data_type | 说明 |
|---------|-----------|------|
| `exchange_binance_kline.py` | `kline` | Binance 现货/合约 K 线数据采集 |
| `exchange_binance_symbol.py` | `symbol` | Binance 交易对列表同步 |

---

## 二、项目结构

```
data-collector/
├── cmd/
│   └── serverless/main.go          # Go 主入口，初始化 HTTPPluginAdapter + scf-framework
│
├── plugin/
│   ├── main.py                      # Python 插件框架：HTTP 服务 + 自动发现 + 任务分发
│   ├── exchange_binance_kline.py    # Binance K线采集插件
│   ├── exchange_binance_symbol.py   # Binance 交易对同步插件
│   └── requirements.txt            # Python 依赖说明
│
├── configs/
│   ├── config.yaml                  # scf-framework 配置（触发器、心跳、DNS 代理、CLS）
│   └── trpc_go.yaml                 # TRPC Server 配置（端口、Timer、日志）
│
├── scf_bootstrap                    # SCF 启动脚本（先启 Python，再启 Go）
├── Makefile                         # 构建和部署
└── go.mod
```

---

## 三、整体架构

```
┌──────────────────────────────────────────────────────────────┐
│                      Moox Server（服务端）                     │
│   - 任务调度中心：将采集任务分配到各云函数节点                   │
│   - 心跳管理服务：接收节点心跳，下发任务实例                     │
│   - 任务状态收集：接收采集执行结果（由 Go 框架异步上报）         │
│   - 探测报文下发：提供 server_ip/port、storage_server_url      │
└─────────────┬────────────────────────────────────────────────┘
              │
     心跳 (每9秒) + 任务实例下发
              │
              ▼
┌──────────────────────────────────────────────────────────────┐
│                 腾讯云 SCF 云函数节点                          │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │           Go 进程（端口 9000，scf-framework 驱动）       │  │
│  │                                                          │  │
│  │  • HTTP Gateway: /health, /probe, /* (转发到 Python)     │  │
│  │  • Heartbeat Reporter: 每 9 秒上报节点状态               │  │
│  │  • TriggerManager:                                       │  │
│  │      - scheduled-collect (cron: 0 * * * * * *)           │  │
│  │      - 触发前：invalid 过滤 + should_execute 周期判断    │  │
│  │      - 触发后：解析响应 → 上报 task_results + 写入数据   │  │
│  │  • DNS Proxy: 定时 DNS 解析 + HTTPS/TCP 探测排序         │  │
│  │  • TaskInstanceStore: 内存缓存任务实例（心跳回包更新）    │  │
│  │  • Storage RPC: 将 write_groups 写入 xData（tRPC 协议）  │  │
│  │  • TaskReporter: 将 task_results 异步上报 Moox Server    │  │
│  │  • HTTPPluginAdapter → http://127.0.0.1:9001             │  │
│  └──────────────────────────┬──────────────────────────────┘  │
│                             │ HTTP POST /on-trigger            │
│                             │ payload 包含预处理后的 jobs 列表  │
│                             ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │           Python 进程（端口 9001，业务逻辑）             │  │
│  │                                                          │  │
│  │  • HTTP Server: /health, /on-trigger                     │  │
│  │  • 插件自动发现: 扫描 *.py → 注册 COLLECTOR 模块         │  │
│  │  • 任务分发: 按 data_type 路由到对应插件                  │  │
│  │  • DNS 记录: 从 metadata["dns_records"] 获取最优 IP      │  │
│  │  • 采集结果: 返回 task_results + write_groups 给 Go 框架 │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                               │
└───────────────────────────────────────────────────────────────┘
              │                              │
              ▼                              ▼
┌─────────────────────┐      ┌───────────────────────────────┐
│    Moox Server       │      │          xData 存储            │
│ /ReportTaskStatus    │      │  set_data / upsert_object     │
│ (task_results 上报)  │      │  (write_groups 写入，tRPC)    │
└─────────────────────┘      └───────────────────────────────┘
```

---

## 四、插件开发规范

### 4.1 插件自动发现机制

Python 进程启动时，`main.py` 中的 `_discover_collectors()` 会：

1. 扫描 `plugin/` 目录下所有 `*.py` 文件
2. 跳过 `main.py` 和以 `_` 开头的文件
3. 对每个文件执行 `importlib.import_module()`
4. 检查模块是否定义了 `COLLECTOR` 字典变量
5. 将 `COLLECTOR["data_type"]` 作为 key 注册到 `_collector_registry`

**新增插件无需修改 `main.py`**，只需在 `plugin/` 目录下新建 `.py` 文件并按规范导出 `COLLECTOR` 即可。

### 4.2 COLLECTOR 字典规范

每个插件必须在模块顶层定义一个名为 `COLLECTOR` 的字典：

```python
COLLECTOR = {
    "data_type": "kline",              # 必填，唯一标识，用于 job 路由
    "data_source": "binance",          # 可选，标识数据来源
    "collect": collect_klines,         # 必填，采集入口函数
    "parse_job": parse_job,            # 可选，job 解析函数
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `data_type` | `str` | 是 | 唯一标识，框架按此字段将 job 路由到插件 |
| `data_source` | `str` | 否 | 数据来源标识，用于日志和调试 |
| `collect` | `callable` | 是 | 采集入口函数，签名见下文 |
| `parse_job` | `callable` | 否 | job 预处理函数，签名见下文。未提供时 job 原样传入 collect |

### 4.3 函数签名规范

#### `parse_job(job_raw: dict) -> Optional[dict]`

从框架下发的原始 job 中提取插件所需参数。返回 `None` 表示跳过该 job。

```python
def parse_job(job_raw: dict) -> Optional[dict]:
    """
    参数:
        job_raw - 框架下发的原始 job，结构如下:
            {
                "task": {
                    "id": 1,
                    "task_id": "task-001",
                    "task_params": "{\"data_source\":\"binance\",\"data_type\":\"kline\",...}"
                },
                "interval": "1m"
            }

    返回:
        解析后的 dict（传给 collect 函数），或 None（跳过该 job）
    """
    task = job_raw.get("task", {})
    task_id = task.get("task_id", "")
    task_params_raw = task.get("task_params", "")

    try:
        params = json.loads(task_params_raw) if task_params_raw else {}
    except (json.JSONDecodeError, TypeError):
        return None

    # 提取并校验必要参数，缺失则返回 None
    symbol = params.get("symbol", "")
    if not symbol:
        return None

    return {
        "task_id": task_id,
        "symbol": symbol,
        "interval": job_raw.get("interval", ""),
        # ... 其他插件需要的字段
    }
```

#### `collect(jobs: list[dict], get_best_ip: callable) -> dict`

采集入口函数，接收解析后的 job 列表，执行采集逻辑，返回结果。

```python
def collect(jobs: list[dict], get_best_ip) -> dict:
    """
    参数:
        jobs        - parse_job 返回的 dict 列表（如未定义 parse_job，则为原始 job 列表）
        get_best_ip - callable(domain: str) -> Optional[str]
                      获取域名的最优 IP（从框架 DNS 解析结果中读取）

    返回:
        {
            "task_results": [                  # 必填，任务执行结果
                {
                    "task_id": "task-001",
                    "status": 2,               # 2=成功, 4=失败
                    "result": ""               # 成功为空，失败为错误信息
                }
            ],
            "write_groups": [                  # 可选，需要写入 xData 的数据
                {
                    "write_mode": "set_data",  # set_data 或 upsert_object
                    "dataset_id": 100,
                    "freq": "1m",              # set_data 模式需要
                    "app_key": "kline-sync",   # upsert_object 模式需要
                    "data_points": [...]
                }
            ]
        }
    """
```

**返回值说明：**

| 字段 | 必填 | 说明 |
|------|------|------|
| `task_results` | 是 | 任务执行结果列表，框架据此上报任务状态到 Moox Server |
| `write_groups` | 否 | 需要写入存储的数据分组，框架统一调用 xData 存储服务写入 |

**task_results 状态码：**

| status | 含义 | 说明 |
|--------|------|------|
| 2 | SUCCESS | 采集成功 |
| 4 | FAILED | 采集失败，`result` 字段应包含错误信息 |

**write_groups.write_mode：**

| write_mode | 说明 | 必填字段 |
|------------|------|---------|
| `set_data` | 按时间序列写入（K线等时序数据） | `dataset_id`, `freq`, `data_points` |
| `upsert_object` | 按主键更新或插入（交易对列表等维度数据） | `dataset_id`, `app_key`, `data_points` |

### 4.4 Python 编码约束

| 约束项 | 要求 | 原因 |
|--------|------|------|
| Python 版本兼容 | **必须兼容 Python 3.9** | SCF 运行时为 `/var/lang/python39/bin/python3.9` |
| 类型注解 | 使用 `Optional[dict]` 而非 `dict \| None` | `X \| Y` 语法需要 Python 3.10+，在 3.9 上会 SyntaxError 导致模块加载失败 |
| 类型注解 | 使用 `list[dict]` 而非 `List[dict]` | Python 3.9 已支持内置类型的泛型语法 |
| import | 需要 `from typing import Optional` | 用于函数返回值类型标注 |
| 标准库优先 | 尽量使用标准库（`urllib`, `json`, `logging`, `ssl`） | 减少依赖，部署包更小 |
| 异常处理 | `parse_job` 内部捕获异常并返回 `None` | 单个 job 解析失败不应阻断其他 job |
| 日志 | 使用 `logging.getLogger(__name__)` | 自动集成到框架日志系统 |
| 并发 | 推荐使用 `concurrent.futures.ThreadPoolExecutor` | 与现有插件保持一致，`max_workers=min(len(jobs), 10)` |

### 4.5 完整插件示例

以新增一个 OKX K线采集插件为例，创建 `plugin/exchange_okx_kline.py`：

```python
"""
OKX K线数据采集插件
"""
import json
import logging
import ssl
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from typing import Optional
from urllib.request import Request, urlopen

logger = logging.getLogger(__name__)

OKX_BASE = "https://www.okx.com"
_DATASET_IDS = {"SWAP": 200, "SPOT": 201}
STATUS_SUCCESS = 2
STATUS_FAILED = 4


def parse_job(job_raw: dict) -> Optional[dict]:
    """从原始 job 中提取 OKX K线采集参数。"""
    task = job_raw.get("task", {})
    task_id = task.get("task_id", "")
    task_params_raw = task.get("task_params", "")
    try:
        params = json.loads(task_params_raw) if task_params_raw else {}
    except (json.JSONDecodeError, TypeError):
        logger.warning(f"[parse_job] task_params 解析失败: task_id={task_id}")
        return None

    inst_type = params.get("inst_type", "")
    symbol = params.get("symbol", "")
    if not inst_type or not symbol:
        logger.warning(f"[parse_job] 缺少必要参数: task_id={task_id}")
        return None

    return {
        "task_id": task_id,
        "inst_type": inst_type,
        "symbol": symbol,
        "interval": job_raw.get("interval", ""),
    }


def collect_okx_klines(jobs: list[dict], get_best_ip) -> dict:
    """执行 OKX K线采集。"""
    if not jobs:
        return {"task_results": [], "write_groups": []}

    task_errors: dict[str, list[str]] = {}
    task_ids_seen: set[str] = set()
    all_data_points = []

    def _do_collect(job):
        task_id = job["task_id"]
        symbol = job["symbol"]
        interval = job["interval"]
        try:
            klines = _fetch_okx_klines(symbol, interval)
            data_points = _format_data_points(symbol, klines)
            logger.info(f"采集成功: symbol={symbol}, interval={interval}, count={len(klines)}")
            return task_id, True, None, data_points
        except Exception as e:
            logger.error(f"采集失败: symbol={symbol}, interval={interval}, error={e}")
            return task_id, False, str(e), []

    with ThreadPoolExecutor(max_workers=min(len(jobs), 10)) as executor:
        futures = {executor.submit(_do_collect, job): job for job in jobs}
        for future in as_completed(futures):
            task_id, ok, err_msg, data_points = future.result()
            task_ids_seen.add(task_id)
            if ok:
                all_data_points.extend(data_points)
            else:
                task_errors.setdefault(task_id, []).append(err_msg)

    # 构建 task_results
    task_results = []
    for task_id in task_ids_seen:
        if task_id in task_errors:
            task_results.append({"task_id": task_id, "status": STATUS_FAILED, "result": "; ".join(task_errors[task_id])})
        else:
            task_results.append({"task_id": task_id, "status": STATUS_SUCCESS, "result": ""})

    # 构建 write_groups
    write_groups = []
    if all_data_points and jobs:
        write_groups.append({
            "write_mode": "set_data",
            "dataset_id": _DATASET_IDS.get(jobs[0]["inst_type"], 200),
            "freq": jobs[0].get("interval", "1m"),
            "data_points": all_data_points,
        })

    return {"task_results": task_results, "write_groups": write_groups}


# ---- 注册插件 ----
COLLECTOR = {
    "data_type": "okx_kline",
    "data_source": "okx",
    "collect": collect_okx_klines,
    "parse_job": parse_job,
}


# ---- 内部函数 ----

def _fetch_okx_klines(symbol: str, interval: str, limit: int = 5) -> list[dict]:
    """从 OKX API 获取 K线数据。"""
    inst_id = symbol.replace("-", "-")  # OKX 格式: BTC-USDT
    url = f"{OKX_BASE}/api/v5/market/candles?instId={inst_id}&bar={interval}&limit={limit}"
    req = Request(url)
    req.add_header("User-Agent", "data-collector/1.0")
    resp = urlopen(req, timeout=10)
    raw = json.loads(resp.read().decode())
    # ... 解析为标准化结构 ...
    return []


def _format_data_points(symbol: str, klines: list[dict]) -> list[dict]:
    """将 K线数据格式化为 xData DataPoint。"""
    # ... 格式化逻辑 ...
    return []
```

### 4.6 新增插件检查清单

- [ ] 文件放在 `plugin/` 目录下，文件名不以 `_` 开头
- [ ] 模块顶层定义 `COLLECTOR` 字典，包含 `data_type`（唯一）和 `collect` 函数
- [ ] `collect` 函数签名为 `(jobs: list[dict], get_best_ip) -> dict`
- [ ] 返回值包含 `task_results` 列表（每个 task_id 对应一条结果）
- [ ] 使用 `from typing import Optional`，不使用 `dict | None` 语法
- [ ] 代码兼容 Python 3.9
- [ ] `parse_job` 中异常时返回 `None` 而非抛出异常
- [ ] 在 `configs/config.yaml` 的 `plugin.supported_collectors` 中添加新的 `data_type`
- [ ] 在 Moox Server 中创建对应的任务实例（`task_params.data_type` 匹配插件的 `data_type`）

---

## 五、启动流程

### 5.1 SCF 启动脚本 (`scf_bootstrap`)

```bash
export PORT=9000
# 1. 后台启动 Python 插件进程
/var/lang/python39/bin/python3.9 -u ./plugin/main.py &
# 2. 前台启动 Go 主进程
./main --conf ./configs/trpc_go.yaml
```

### 5.2 Go 主进程启动 (`cmd/serverless/main.go`)

```go
collectors := loadSupportedCollectors("./configs/config.yaml")

p := plugin.NewHTTPPluginAdapter(
    "data-collector",
    "http://127.0.0.1:9001",
    plugin.WithReadyTimeout(60*time.Second),
    plugin.WithHeartbeatExtra(map[string]interface{}{
        "supported_collectors": collectors,
    }),
)

app := scf.New(p,
    scf.WithConfigPath("./configs/config.yaml"),
    scf.WithGatewayService("trpc.collector.gateway.stdhttp"),
)

app.Run(trpc.BackgroundContext())
```

启动序列：

1. 加载 `configs/config.yaml`，读取 `plugin.supported_collectors`
2. 创建 TRPC Server（端口 9000）
3. 初始化 RuntimeState（从环境变量 `SCF_FUNCTIONNAME` 读取 NodeID）
4. 初始化 TaskInstanceStore
5. HTTPPluginAdapter 轮询 `GET http://127.0.0.1:9001/health`，等待 Python 就绪（最长 60 秒）
6. 注册 HTTP Gateway（含 catch-all 转发到 Python）
7. 注册心跳 Timer（每 9 秒上报）
8. 初始化 DNS Proxy（定时 DNS 解析 + IP 探测，结果注入 metadata）
9. 初始化触发器 `scheduled-collect`（cron: `0 * * * * * *`，每分钟整点）
10. 启动 TRPC Server

### 5.3 Python 进程启动 (`plugin/main.py`)

1. 初始化控制台日志
2. 加载 CLS 日志 handler（从 `configs/config.yaml` 的 `plugin.cls` 节点读取配置）
3. 调用 `_discover_collectors()` 扫描并注册所有插件
4. 启动 HTTP Server 监听 `0.0.0.0:9001`

---

## 六、触发器与调度

### 6.1 触发事件处理

当 `scheduled-collect` 触发时，scf-framework 先在 Go 层完成任务预处理（invalid 过滤、should_execute 周期判断），然后向 Python 进程发送 `POST /on-trigger`：

```json
{
    "type": "timer",
    "name": "scheduled-collect",
    "payload": {
        "jobs": [
            {
                "task": {
                    "id": 1,
                    "task_id": "task-001",
                    "task_params": "{\"data_source\":\"binance\",\"data_type\":\"kline\",\"inst_type\":\"SPOT\",\"symbol\":\"BTC-USDT\",\"intervals\":[\"1m\",\"5m\",\"1h\"]}"
                },
                "interval": "1m"
            }
        ]
    },
    "metadata": {
        "nodeID": "node-abc",
        "version": "v0.0.5",
        "dns_records": "{...}",
        "storage_server_url": "http://xx.xx.xx.xx:19104"
    }
}
```

### 6.2 Job 分发流程

```
POST /on-trigger
    │
    ▼
handle_trigger(event)
    ├── 更新日志上下文（nodeID, version）
    ├── 更新 DNS 记录缓存
    │
    ▼
_dispatch_jobs(payload)
    │
    ├── 遍历 jobs，解析 task_params
    ├── 按 data_type 分组: grouped[data_type] = [job1, job2, ...]
    │
    ├── 对每个 data_type:
    │   ├── 查找 _collector_registry[data_type]
    │   ├── 调用 parse_job(job_raw) → 过滤 None
    │   ├── 调用 collect(parsed_jobs, get_best_ip)
    │   └── 收集 task_results + write_groups
    │
    ▼
返回聚合结果给 scf-framework
```

### 6.3 采集周期判断（Go 框架层）

框架层的 `should_execute()` 根据当前 UTC 时间判断是否执行指定周期：

| 周期 | 规则 | 示例 |
|------|------|------|
| `1m` | 每分钟执行 | 始终执行 |
| `5m` | minute % 5 == 0 | 00:00, 00:05, 00:10... |
| `15m` | minute % 15 == 0 | 00:00, 00:15, 00:30, 00:45 |
| `1h` | minute == 0 | 整点执行 |
| `4h` | minute == 0 且 hour % 4 == 0 | 00:00, 04:00, 08:00... |
| `1d` | hour == 0 且 minute == 0 | 每天 00:00 UTC |
| `1w` | 周一 00:00 UTC | 每周一 |
| `1M` | 每月 1 号 00:00 UTC | 月初 |

---

## 七、DNS 优化

### 7.1 工作原理

DNS 解析由 Go 框架层（`dns_proxy` 配置 + `trpc.dns.timer`）统一管理，Python 插件通过 metadata 被动接收：

```
trpc.dns.timer（每分钟第 30 秒触发）
    │
    ▼
向多个 DNS 服务器并发解析目标域名
    │
    ▼
对解析出的 IP 执行 HTTPS 探测（按 probe_configs）
    │
    ▼
按 (可用性, 延迟) 排序，缓存为 dns_records
    │
    ▼
触发事件时通过 metadata["dns_records"] 注入 Python
    │
    ▼
Python 端 get_best_ip(domain) → 延迟最低的可用 IP
```

### 7.2 IP 替换机制

当有最优 IP 时，采集函数会：

1. 将 URL 中的域名替换为 IP 地址
2. 设置 `Host` header 为原始域名
3. 禁用 SSL 证书校验（因为证书绑定域名而非 IP）

插件中使用 DNS 优化：

```python
def _do_collect(job):
    domain = "api.binance.com"
    best_ip = get_best_ip(domain)     # 由框架注入的函数
    # best_ip 可能为 None（无可用 IP 时回退域名直连）
```

---

## 八、任务管理

### 8.1 任务实例生命周期

```
Moox Server
    │
    ├── ① 创建采集任务规则
    ├── ② 将任务实例分配到节点
    │
    ▼
③ 心跳响应中下发 task_instances
    │
    ▼
TaskInstanceStore (Go 内存)
    │
    ├── ④ 触发时预处理：过滤 invalid、周期判断、生成 jobs
    │
    ▼
Python 插件按 data_type 分发到对应采集插件
    │
    ├── ⑤ 执行采集
    │
    ▼
⑥ 返回 task_results + write_groups → scf-framework 异步上报
```

### 8.2 任务参数结构 (task_params)

```json
{
    "data_source": "binance",       // 数据源标识
    "data_type": "kline",           // 数据类型（对应插件 COLLECTOR.data_type）
    "inst_type": "SPOT",            // 产品类型
    "symbol": "BTC-USDT",           // 交易对
    "intervals": ["1m", "5m", "1h"] // 采集周期列表
}
```

### 8.3 任务执行结果汇总规则

同一个 `task_id` 下的所有 interval 采集中，只要有一个失败，整个 task 标记为失败，`result` 字段包含所有失败的错误信息（分号分隔）。

---

## 九、心跳与监控

### 9.1 心跳上报

- **间隔**: 9 秒（TRPC Timer 驱动）
- **上报地址**: `POST http://{server_ip}:{server_port}/gateway/cloudnode/ReportHeartbeatInner`
- **携带数据**: node_id, node_type, running_version, tasks_md5, supported_collectors

### 9.2 心跳响应

| 字段 | 说明 |
|------|------|
| `package_version` | 服务端期望版本。若与本地不一致，Go 进程 Fatal 退出（由 SCF 平台拉起新版本） |
| `task_instances` | 新的任务实例列表（仅当 tasks_md5 不匹配时下发） |

### 9.3 健康检查

| 端点 | 说明 |
|------|------|
| `GET :9000/health` | Go Gateway 健康检查 |
| `GET :9001/health` | Python 插件健康检查 |
| `POST :9000/probe` | 服务端探测请求（下发 server_ip/port、storage_server_url） |

---

## 十、构建与部署

### 10.1 构建

```bash
make build-scf v0.0.5
```

构建过程：
1. `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build` 编译 Go 二进制
2. 复制配置文件，将 `config.yaml` 中的 version 更新为指定版本号
3. 复制 Python 插件（`plugin/*.py`）
4. 复制 scf_log 模块（从 `../scf-framework/python/scf_log/`）
5. 安装 Python 依赖（pip install --target=plugin/，指定 linux x86_64 + Python 3.9）
6. 打包为 `collector-scf-v0.0.5.zip`

### 10.2 部署

```bash
make deploy SERVER=ubuntu@143.177.177.177
```

### 10.3 部署包结构

```
collector-scf-v0.0.5.zip
├── main                          # Go 二进制 (linux/amd64)
├── scf_bootstrap                 # 启动脚本
├── configs/
│   ├── config.yaml
│   └── trpc_go.yaml
└── plugin/
    ├── main.py                   # 插件框架
    ├── exchange_binance_kline.py # Binance K线插件
    ├── exchange_binance_symbol.py# Binance 交易对插件
    ├── scf_log/                  # CLS 日志模块
    └── (pip 依赖)
```
