// Package services 数据采集器服务实现（最好是中文注释！）
package services

import (
	"context"

	pb "github.com/mooyang-code/data-collector/proto/gen"
	"trpc.group/trpc-go/trpc-go/log"
)

// CollectorServiceImpl 数据采集器服务实现
type CollectorServiceImpl struct {
	pb.UnimplementedCollector
}

// NewCollectorService 创建新的数据采集器服务实例
func NewCollectorService() *CollectorServiceImpl {
	return &CollectorServiceImpl{}
}

// Empty 空接口实现
func (s *CollectorServiceImpl) Empty(ctx context.Context, req *pb.EmptyReq) (*pb.EmptyRsp, error) {
	log.InfoContext(ctx, "收到Empty请求")
	
	// 这里可以添加您的业务逻辑
	// 例如：健康检查、状态查询等
	
	return &pb.EmptyRsp{}, nil
}
