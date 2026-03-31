package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/Sandwichzzy/market-services/common/httputil"
	"github.com/Sandwichzzy/market-services/config"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/Sandwichzzy/market-services/services/http/routes"
	"github.com/Sandwichzzy/market-services/services/http/service"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

const (
	HealthPath       = "healthz"
	SupportAssetPath = "/api/v1/get_support_assets"
)

// APIConfig API 服务配置
// 包含 HTTP 服务器和监控指标服务器的配置
type ApiConfig struct {
	HttpServer   config.ServerConfig // HTTP API 服务器配置（端口、主机等）
	MetricServer config.ServerConfig // 监控指标服务器配置
}

// API 核心结构体，管理整个 HTTP API 服务的生命周期
type API struct {
	router  *chi.Mux
	apiSvr  *httputil.HTTPServer
	db      *database.DB
	stopped atomic.Bool
}

// NewAPI 创建并初始化一个新的 API 实例
// 参数：
//   - ctx: 上下文，用于控制初始化过程
//   - cfg: 配置对象，包含数据库、服务器、缓存等所有配置
//
// 返回：
//   - *API: 初始化成功的 API 实例
//   - error: 如果初始化失败，会返回错误并自动清理已创建的资源
//
// 初始化流程：
//  1. 初始化数据库连接
//  2. 初始化路由和中间件
//  3. 启动 HTTP 服务器
func NewApi(ctx context.Context, cfg *config.Config) (*API, error) {
	out := &API{}
	if err := out.initFromConfig(ctx, cfg); err != nil {
		return nil, errors.Join(err, out.Stop(ctx))
	}
	return out, nil
}

// initFromConfig 从配置文件初始化 API 服务的所有组件
//
// 参数：
//   - ctx: 上下文
//   - cfg: 配置对象
//
// 返回：
//   - error: 初始化过程中的任何错误
//
// 初始化步骤：
//  1. 初始化数据库连接（主库或从库）
//  2. 初始化路由、中间件和处理器
//  3. 启动 HTTP 服务器
func (a *API) initFromConfig(ctx context.Context, cfg *config.Config) error {
	if err := a.initDB(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init DB: %w", err)
	}
	a.initRouter(cfg.RestServer, cfg)
	if err := a.startServer(cfg.RestServer); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}
	return nil
}

func (a *API) initDB(ctx context.Context, cfg *config.Config) error {
	initDb, err := database.NewDB(ctx, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to slave database", "err", err)
		return err
	}
	a.db = initDb
	return nil
}

func (a *API) initRouter(conf config.ServerConfig, cfg *config.Config) {

	// 创建服务层实例
	svc := service.NewHandleSvc(a.db.Asset)
	// 创建路由器实例
	apiRouter := chi.NewRouter()
	// 创建路由处理器，连接服务层
	h := routes.NewRoutes(apiRouter, svc)

	// 配置中间件
	apiRouter.Use(middleware.Timeout(time.Second * 12))
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(middleware.Heartbeat(HealthPath))

	// 配置 CORS（跨域资源共享）
	apiRouter.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},                                                       // 允许所有来源（生产环境应限制）
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},                 // 允许的 HTTP 方法
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}, // 允许的请求头
		ExposedHeaders:   []string{"Link"},                                                    // 暴露的响应头
		AllowCredentials: true,                                                                // 允许发送凭证
		MaxAge:           300,                                                                 // 预检请求缓存时间（秒）
	}).Handler)

	// 注册 API 路由
	//post API
	apiRouter.Post(fmt.Sprintf(SupportAssetPath), h.GetSupportAssets)

	a.router = apiRouter
}

func (a *API) Start(ctx context.Context) error {
	return nil
}

// Stop 优雅地停止 API 服务
// 参数：
//   - ctx: 上下文，用于控制停止超时
//
// 返回：
//   - error: 停止过程中的任何错误（可能是多个错误的合并）
//
// 停止流程：
//  1. 停止 HTTP API 服务器
//  2. 关闭数据库连接
//  3. 设置 stopped 标志为 true
//
// 注意：
//   - 即使某个组件停止失败，也会继续停止其他组件
//   - 所有错误会被合并并返回
func (a *API) Stop(ctx context.Context) error {
	var result error
	if a.apiSvr != nil {
		if err := a.apiSvr.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop API server: %w", err))
		}
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close DB: %w", err))
		}
	}
	a.stopped.Store(true)
	log.Info("API service shutdown complete")
	return result
}

// startServer 启动 HTTP 服务器
//
// 参数：
//   - serverConfig: 服务器配置（主机和端口）
//
// 返回：
//   - error: 启动失败时返回错误
//
// 功能：
//   - 根据配置的主机和端口启动 HTTP 服务器
//   - 记录服务器监听地址
func (a *API) startServer(serverConfig config.ServerConfig) error {
	log.Debug("API server listening...", "port", serverConfig.Port)
	addr := net.JoinHostPort(serverConfig.Host, strconv.Itoa(serverConfig.Port))
	srv, err := httputil.StartHttpServer(addr, a.router)
	if err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}
	log.Info("API server started", "addr", srv.Addr().String())
	a.apiSvr = srv
	return nil
}

// Stopped 检查 API 服务是否已停止
//
// 返回：
//   - bool: true 表示服务已停止，false 表示服务仍在运行
//
// 用途：
//   - 用于健康检查
//   - 用于优雅关闭流程中的状态判断
func (a *API) Stopped() bool {
	return a.stopped.Load()
}
