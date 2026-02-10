"""
Binance Symbol（标的同步）采集插件

负责从 Binance Spot/Swap ExchangeInfo API 获取交易对列表，
过滤后以 DataPoint 格式返回，由 scf-framework 统一写入 xData（UpsertObject）。
通过 COLLECTOR 自注册机制，由 main.py 自动发现并调度。
"""

import json
import logging
import ssl
from typing import Optional
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError

logger = logging.getLogger("data-collector-plugin")

# ============================================================================
# 配置
# ============================================================================

BINANCE_SPOT_BASE = "https://api.binance.com"
BINANCE_SWAP_BASE = "https://fapi.binance.com"

_EXCHANGE_INFO_CONFIG = {
    "SPOT": (BINANCE_SPOT_BASE, "/api/v3/exchangeInfo", "api.binance.com"),
    "SWAP": (BINANCE_SWAP_BASE, "/fapi/v1/exchangeInfo", "fapi.binance.com"),
}

_DATASET_IDS = {"SWAP": 100, "SPOT": 101}

STATUS_SUCCESS = 2
STATUS_FAILED = 4

# ============================================================================
# 自注册
# ============================================================================


def parse_job(job_raw: dict) -> Optional[dict]:
    """从 framework job 中提取 symbol 采集所需参数。返回 None 表示跳过。"""
    task = job_raw.get("task", {})
    task_id = task.get("task_id", "")
    task_params_raw = task.get("task_params", "")
    try:
        params = json.loads(task_params_raw) if task_params_raw else {}
    except (json.JSONDecodeError, TypeError):
        return None

    inst_type = params.get("inst_type", "")
    if not inst_type:
        return None

    return {
        "task_id": task_id,
        "inst_type": inst_type,
    }


def collect_symbols(jobs: list[dict], get_best_ip) -> dict:
    """执行 symbol 采集（串行）。

    Args:
        jobs: 已解析的 job 列表，每个 job 为 parse_job 返回的 dict
        get_best_ip: callable(domain) -> Optional[str]，获取最优 IP

    Returns:
        {"task_results": [...], "write_groups": [...]}
    """
    if not jobs:
        return {"task_results": [], "write_groups": []}

    logger.info(f"本轮 symbol 采集: {len(jobs)} 个任务")

    task_results = []
    all_data_points = []
    last_inst_type = "SWAP"

    for job in jobs:
        task_id = job["task_id"]
        inst_type = job["inst_type"]
        last_inst_type = inst_type
        try:
            domain = _get_domain(inst_type)
            best_ip = get_best_ip(domain) if domain else None
            raw_symbols = _fetch_symbols(inst_type, best_ip=best_ip)
            filtered = _filter_symbols(raw_symbols, inst_type)
            data_points = _format_data_points(filtered)
            all_data_points.extend(data_points)
            logger.info(f"Symbol 采集成功: taskID={task_id}, instType={inst_type}, count={len(filtered)}")
            task_results.append({
                "task_id": task_id,
                "status": STATUS_SUCCESS,
                "result": "",
            })
        except Exception as e:
            logger.error(f"Symbol 采集失败: taskID={task_id}, instType={inst_type}, error={e}")
            task_results.append({
                "task_id": task_id,
                "status": STATUS_FAILED,
                "result": str(e),
            })

    write_groups = []
    if all_data_points:
        write_groups.append({
            "write_mode": "upsert_object",
            "dataset_id": _DATASET_IDS.get(last_inst_type, 100),
            "app_key": "symbol-sync",
            "data_points": all_data_points,
        })

    return {
        "task_results": task_results,
        "write_groups": write_groups,
    }


COLLECTOR = {
    "data_type": "symbol",
    "data_source": "binance",
    "collect": collect_symbols,
    "parse_job": parse_job,
}

# ============================================================================
# 内部函数
# ============================================================================


def _fetch_symbols(inst_type: str, best_ip: Optional[str] = None) -> list[dict]:
    """从 Binance ExchangeInfo API 获取交易对列表。"""
    cfg = _EXCHANGE_INFO_CONFIG.get(inst_type)
    if cfg is None:
        raise ValueError(f"不支持的产品类型: {inst_type}")

    base_url, api_path, domain = cfg
    url = f"{base_url}{api_path}"

    req = Request(url)
    req.add_header("User-Agent", "data-collector/1.0")

    try:
        if best_ip:
            url_with_ip = url.replace(f"https://{domain}", f"https://{best_ip}")
            req = Request(url_with_ip)
            req.add_header("Host", domain)
            req.add_header("User-Agent", "data-collector/1.0")

            ctx = ssl.create_default_context()
            ctx.check_hostname = False
            ctx.verify_mode = ssl.CERT_NONE
            resp = urlopen(req, timeout=30, context=ctx)
        else:
            resp = urlopen(req, timeout=30)

        raw = json.loads(resp.read().decode())
    except (URLError, HTTPError) as e:
        logger.error(f"Binance ExchangeInfo API 请求失败: inst_type={inst_type}, error={e}")
        raise

    symbols = raw.get("symbols", [])
    logger.info(f"ExchangeInfo 获取完成: inst_type={inst_type}, total_symbols={len(symbols)}")
    return symbols


def _filter_symbols(symbols: list[dict], inst_type: str) -> list[dict]:
    """过滤并标准化交易对。

    过滤条件：
      - status == "TRADING" 且 quoteAsset == "USDT"
      - SWAP 额外要求 contractType == "PERPETUAL"
    输出标准化格式: symbol 字段为 "BTC-USDT" 形式。
    """
    result = []
    for s in symbols:
        if s.get("status", "") != "TRADING":
            continue
        if s.get("quoteAsset", "") != "USDT":
            continue
        if inst_type == "SWAP" and s.get("contractType", "") != "PERPETUAL":
            continue

        base_asset = s.get("baseAsset", "")
        if not base_asset:
            continue

        result.append({"symbol": f"{base_asset}-{s['quoteAsset']}"})

    logger.info(f"Symbol 过滤完成: inst_type={inst_type}, "
                f"before={len(symbols)}, after={len(result)}")
    return result


def _format_data_points(symbols: list[dict]) -> list[dict]:
    """将 symbol 列表转为框架 DataPoint 格式。"""
    data_points = []
    for sym in symbols:
        data_points.append({
            "times": "",
            "object_id": sym["symbol"],
            "fields": {
                "symbol": sym["symbol"],
                "unshelve_time": "2099-01-01 00:00:00",
            },
        })
    return data_points


def _get_domain(inst_type: str) -> Optional[str]:
    """返回指定产品类型对应的 Binance API 域名。"""
    cfg = _EXCHANGE_INFO_CONFIG.get(inst_type)
    return cfg[2] if cfg else None
