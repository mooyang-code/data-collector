// Package main trpc客户端测试程序（最好是中文注释！）
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	pb "github.com/mooyang-code/data-collector/proto/gen"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
)

var (
	serverAddr = flag.String("addr", "127.0.0.1:18401", "服务器地址")
)

func main() {
	flag.Parse()

	fmt.Printf("连接到 trpc 服务器: %s\n", *serverAddr)

	// 创建客户端代理
	proxy := pb.NewCollectorClientProxy(
		client.WithTarget("ip://" + *serverAddr),
		client.WithTimeout(3*time.Second),
	)

	// 创建请求上下文
	ctx := context.Background()

	// 调用Empty接口
	req := &pb.EmptyReq{}
	rsp, err := proxy.Empty(ctx, req)
	if err != nil {
		log.Errorf("调用Empty接口失败: %v", err)
		return
	}

	fmt.Printf("调用成功，响应: %+v\n", rsp)
	fmt.Println("✅ trpc 服务测试完成")
}
