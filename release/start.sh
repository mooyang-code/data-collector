#!/bin/bash

# Data Collector 启动脚本

APP_NAME="data-collector"
PID_FILE="./$APP_NAME.pid"

# 检查并停止已存在的进程
echo "检查已存在的进程..."

# 检查PID文件中的进程
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if ps -p "$OLD_PID" > /dev/null 2>&1; then
        echo "发现运行中的服务 (PID: $OLD_PID)，正在停止..."
        kill "$OLD_PID"

        # 等待进程结束
        for i in {1..10}; do
            if ! ps -p "$OLD_PID" > /dev/null 2>&1; then
                echo "旧进程已停止"
                break
            fi
            sleep 1
        done

        # 如果还在运行，强制杀死
        if ps -p "$OLD_PID" > /dev/null 2>&1; then
            echo "强制停止旧进程..."
            kill -9 "$OLD_PID"
        fi
    fi
    rm -f "$PID_FILE"
fi

# 通过进程名查找并停止可能的进程
RUNNING_PIDS=$(pgrep -f "$APP_NAME" 2>/dev/null || true)
if [ ! -z "$RUNNING_PIDS" ]; then
    echo "发现其他运行中的 $APP_NAME 进程: $RUNNING_PIDS"
    echo "正在停止这些进程..."
    echo "$RUNNING_PIDS" | xargs kill 2>/dev/null || true
    sleep 2

    # 强制杀死仍在运行的进程
    STILL_RUNNING=$(pgrep -f "$APP_NAME" 2>/dev/null || true)
    if [ ! -z "$STILL_RUNNING" ]; then
        echo "强制停止残留进程: $STILL_RUNNING"
        echo "$STILL_RUNNING" | xargs kill -9 2>/dev/null || true
    fi
fi

# 启动服务
echo "启动 $APP_NAME..."
cd ./bin
nohup ./$APP_NAME > ../log/app.log 2>&1 &
echo $! > "../$PID_FILE"
echo "服务已启动 (PID: $(cat ../$PID_FILE))"
echo "日志文件: log/app.log"
