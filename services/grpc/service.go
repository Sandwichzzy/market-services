package grpc

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/services/grpc/proto"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const MaxRecvMessageSize = 1024 * 1024 * 30000

type MarketRpcConfig struct {
	Host string
	Port int
}

type MarketRpcService struct {
	*MarketRpcConfig //匿名字段 嵌入字段（提升）
	db               *database.DB

	proto.UnimplementedMarketServicesServer
	proto.UnimplementedMarketSymbolServiceServer
	proto.UnimplementedFiatCurrencyServiceServer
	proto.UnimplementedKlineServiceServer
	stopped atomic.Bool
}

func NewMarketRpcService(conf *MarketRpcConfig, db *database.DB) (*MarketRpcService, error) {
	return &MarketRpcService{
		MarketRpcConfig: conf,
		db:              db,
	}, nil
}

func (ms *MarketRpcService) Start(ctx context.Context) error {
	go func(ms *MarketRpcService) {
		// 构造监听地址，例如 "0.0.0.0:8080"
		rpcAddr := fmt.Sprintf("%s:%d", ms.Host, ms.Port)
		listener, err := net.Listen("tcp", rpcAddr)
		if err != nil {
			log.Error("Failed to start tcp listener", "error", err)
			return
		}
		// 设置 gRPC 服务器选项：最大接收消息大小
		opt := grpc.MaxRecvMsgSize(MaxRecvMessageSize)
		// 创建 gRPC 服务器，可以配置拦截器等
		gs := grpc.NewServer(opt, grpc.ChainUnaryInterceptor(nil))
		// 注册 gRPC 反射服务，允许使用 grpcurl 等工具动态调用接口
		reflection.Register(gs)
		// 将当前服务实现注册到 gRPC 服务器
		// proto.MarketServicesServer 接口由嵌入的 UnimplementedMarketServicesServer 自动满足
		proto.RegisterMarketServicesServer(gs, ms)
		proto.RegisterMarketSymbolServiceServer(gs, ms)
		proto.RegisterFiatCurrencyServiceServer(gs, ms)
		proto.RegisterKlineServiceServer(gs, ms)
		log.Info("grpc info", "addr", listener.Addr())

		// 启动服务并阻塞，直到 Serve 返回
		if err := gs.Serve(listener); err != nil {
			log.Error("start rpc server fail", "err", err)
		}
	}(ms)
	return nil
}

// Stop 停止 gRPC 服务
//
// 注意：此方法仅设置停止标志，并不主动关闭 gRPC 服务器连接。
// 实际服务停止需要依赖外部机制（如信号处理）来关闭 listener，从而让 gs.Serve 退出。
func (ms *MarketRpcService) Stop(ctx context.Context) error {
	ms.stopped.Store(true) // 原子性地将停止标志设为 true
	return nil
}

// Stopped 返回服务是否已被标记为停止
func (ms *MarketRpcService) Stopped() bool {
	return ms.stopped.Load()
}
