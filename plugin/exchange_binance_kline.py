"""
Binance K线采集插件

负责从 Binance Spot/Swap API 获取 K线数据。
通过 COLLECTOR 自注册机制，由 main.py 自动发现并调度。
采集结果以 DataPoint 格式返回，由 scf-framework 统一写入 xData。
"""

import json
import logging
import ssl
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from typing import Optional
from urllib.request import Request, urlopen
from urllib.error import URLError, HTTPError
from urllib.parse import quote

logger = logging.getLogger("data-collector-plugin")

# ============================================================================
# 配置
# ============================================================================

BINANCE_SPOT_BASE = "https://api.binance.com"
BINANCE_SWAP_BASE = "https://fapi.binance.com"

_EXCHANGE_CONFIG = {
    "SPOT": (BINANCE_SPOT_BASE, "/api/v3/klines", "api.binance.com"),
    "SWAP": (BINANCE_SWAP_BASE, "/fapi/v1/klines", "fapi.binance.com"),
}

_DATASET_IDS = {"SWAP": 100, "SPOT": 101}

STATUS_SUCCESS = 2
STATUS_FAILED = 4

# ============================================================================
# 自注册
# ============================================================================


def parse_job(job_raw: dict) -> Optional[dict]:
    """从 framework job 中提取 kline 采集所需参数。返回 None 表示跳过。"""
    task = job_raw.get("task", {})
    task_id = task.get("task_id", "")
    task_params_raw = task.get("task_params", "")
    try:
        params = json.loads(task_params_raw) if task_params_raw else {}
    except (json.JSONDecodeError, TypeError):
        logger.warning(f"[parse_job] task_params 解析失败: task_id={task_id}, raw={task_params_raw[:200]}")
        return None

    inst_type = params.get("inst_type", "")
    symbol = params.get("symbol", "")
    if not inst_type or not symbol:
        logger.warning(f"[parse_job] 缺少必要参数: task_id={task_id}, inst_type={inst_type!r}, symbol={symbol!r}")
        return None

    interval = job_raw.get("interval", "")
    if not interval:
        logger.warning(f"[parse_job] interval 为空: task_id={task_id}, symbol={symbol}")

    return {
        "task_id": task_id,
        "inst_type": inst_type,
        "symbol": symbol,
        "interval": interval,
    }


def collect_klines(jobs: list[dict], get_best_ip) -> dict:
    """执行 kline 采集（并发）。

    Args:
        jobs: 已解析的 job 列表，每个 job 为 parse_job 返回的 dict
        get_best_ip: callable(domain) -> Optional[str]，获取最优 IP

    Returns:
        {"task_results": [...], "write_groups": [...]}
    """
    if not jobs:
        return {"task_results": [], "write_groups": []}

    symbols_desc = ", ".join(f"{j['symbol']}/{j['inst_type']}/{j['interval']}" for j in jobs)
    logger.info(f"本轮 kline 采集: {len(jobs)} 个任务, 标的: [{symbols_desc}]")

    task_errors: dict[str, list[str]] = {}
    task_ids_seen: set[str] = set()
    all_data_points = []

    def _do_collect(job):
        task_id = job["task_id"]
        inst_type = job["inst_type"]
        symbol = job["symbol"]
        interval = job["interval"]
        try:
            domain = _get_domain(inst_type)
            best_ip = get_best_ip(domain) if domain else None
            klines = _fetch_klines(inst_type, symbol, interval, best_ip=best_ip)
            data_points = _format_data_points(symbol, klines)
            logger.info(f"采集成功: symbol={symbol}, instType={inst_type}, interval={interval}, count={len(klines)}")
            return task_id, symbol, interval, True, None, data_points
        except Exception as e:
            logger.error(f"采集失败: symbol={symbol}, instType={inst_type}, interval={interval}, error={e}")
            return task_id, symbol, interval, False, str(e), []

    with ThreadPoolExecutor(max_workers=min(len(jobs), 10)) as executor:
        futures = {executor.submit(_do_collect, job): job for job in jobs}
        succeeded = []
        failed = []
        for future in as_completed(futures):
            task_id, symbol, interval, ok, err_msg, data_points = future.result()
            task_ids_seen.add(task_id)
            if ok:
                succeeded.append(f"{symbol}/{interval}")
                all_data_points.extend(data_points)
            else:
                failed.append(f"{symbol}/{interval}({err_msg})")
                if task_id not in task_errors:
                    task_errors[task_id] = []
                task_errors[task_id].append(f"{symbol}/{interval}: {err_msg}")

    logger.info(f"本轮 kline 采集完成: 成功={len(succeeded)}, 失败={len(failed)}"
                f", 成功标的=[{', '.join(succeeded)}]"
                + (f", 失败标的=[{', '.join(failed)}]" if failed else ""))

    # 构建 task_results
    task_results = []
    for task_id in task_ids_seen:
        if task_id in task_errors:
            task_results.append({
                "task_id": task_id,
                "status": STATUS_FAILED,
                "result": "; ".join(task_errors[task_id]),
            })
        else:
            task_results.append({
                "task_id": task_id,
                "status": STATUS_SUCCESS,
                "result": "",
            })

    # 构建 write_groups
    write_groups = []
    if all_data_points and jobs:
        first = jobs[0]
        interval = first.get("interval", "1m")
        if interval.endswith("h"):
            freq = interval[:-1] + "H"
        elif interval.endswith("d"):
            freq = interval[:-1] + "D"
        else:
            freq = interval
        write_groups.append({
            "write_mode": "set_data",
            "dataset_id": _DATASET_IDS.get(first["inst_type"], 100),
            "freq": freq,
            "data_points": all_data_points,
        })

    return {
        "task_results": task_results,
        "write_groups": write_groups,
    }


COLLECTOR = {
    "data_type": "kline",
    "data_source": "binance",
    "collect": collect_klines,
    "parse_job": parse_job,
}

# ============================================================================
# 内部函数
# ============================================================================


def _fetch_klines(
    inst_type: str,
    symbol: str,
    interval: str,
    limit: int = 5,
    best_ip: Optional[str] = None,
) -> list[dict]:
    """从 Binance API 获取 K线数据。"""
    cfg = _EXCHANGE_CONFIG.get(inst_type)
    if cfg is None:
        raise ValueError(f"不支持的产品类型: {inst_type}")

    base_url, api_path, domain = cfg
    api_symbol = symbol.replace("-", "")
    url = f"{base_url}{api_path}?symbol={quote(api_symbol)}&interval={quote(interval)}&limit={limit}"

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
            resp = urlopen(req, timeout=10, context=ctx)
        else:
            resp = urlopen(req, timeout=10)

        raw = json.loads(resp.read().decode())
    except (URLError, HTTPError) as e:
        logger.error(f"Binance API 请求失败: {e}")
        raise

    klines = []
    for item in raw:
        klines.append({
            "open_time": datetime.fromtimestamp(item[0] / 1000, tz=timezone.utc).strftime("%Y-%m-%d %H:%M:%S"),
            "open": float(item[1]),
            "high": float(item[2]),
            "low": float(item[3]),
            "close": float(item[4]),
            "volume": float(item[5]),
            "close_time": datetime.fromtimestamp(item[6] / 1000, tz=timezone.utc).strftime("%Y-%m-%d %H:%M:%S"),
            "quote_volume": float(item[7]),
            "trade_count": int(item[8]),
        })
    return klines


def _format_data_points(symbol: str, klines: list[dict]) -> list[dict]:
    """将 K线数据转为框架 DataPoint 格式。"""
    data_points = []
    for kline in klines:
        data_points.append({
            "times": kline["open_time"],
            "object_id": symbol,
            "fields": {
                "candle_begin_time": kline["open_time"],
                "candle_end_time": kline["close_time"],
                "open": kline["open"],
                "high": kline["high"],
                "low": kline["low"],
                "close": kline["close"],
                "volume": kline["volume"],
                "quote_volume": kline["quote_volume"],
                "trade_num": kline["trade_count"],
            },
        })
    return data_points


def _get_domain(inst_type: str) -> Optional[str]:
    """返回指定产品类型对应的 Binance API 域名。"""
    cfg = _EXCHANGE_CONFIG.get(inst_type)
    return cfg[2] if cfg else None
