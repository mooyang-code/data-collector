#!/bin/bash

# Data Collector 停止脚本

APP_NAME="data-collector"
PID_FILE="./$APP_NAME.pid"

if [ ! -f "$PID_FILE" ]; then
    echo "PID文件不存在，服务可能没有运行"
    exit 1
fi

PID=$(cat "$PID_FILE")

if ps -p "$PID" > /dev/null 2>&1; then
    echo "停止服务 (PID: $PID)..."
    kill "$PID"

    # 等待进程结束
    for i in {1..10}; do
        if ! ps -p "$PID" > /dev/null 2>&1; then
            echo "服务已停止"
            rm -f "$PID_FILE"
            exit 0
        fi
        sleep 1
    done

    # 强制杀死进程
    echo "强制停止服务..."
    kill -9 "$PID"
    rm -f "$PID_FILE"
else
    echo "服务没有运行"
    rm -f "$PID_FILE"
fi
