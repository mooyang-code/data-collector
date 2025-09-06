#!/bin/bash

echo "启动数据采集器..."
./bin/collector --config configs/config.yaml &
PID=$!

echo "PID: $PID"
echo "等待15秒以观察输出..."
sleep 15

echo "停止采集器..."
kill -SIGINT $PID

echo "等待进程退出..."
wait $PID

echo "测试完成"