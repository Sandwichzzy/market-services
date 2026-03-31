// Package httputil 提供 HTTP 服务的封装工具
// 主要目标：
// 1. 简化 http.Server 启动流程
// 2. 提供优雅关闭（graceful shutdown）能力
// 3. 支持可扩展配置（Option 模式）
// 4. 封装生命周期状态（是否关闭）
package httputil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/pkg/errors"
)

// HTTPServer 是对 http.Server 的二次封装
// 核心能力：
// - 持有 listener，支持获取监听地址
// - 提供关闭状态标识
// - 支持优雅关闭和强制关闭
type HTTPServer struct {
	listener net.Listener // 底层 TCP 监听器
	srv      *http.Server // 标准库 HTTP 服务
	closed   atomic.Bool  // 标记服务是否已关闭（线程安全）
}

// HTTPOption 定义函数式配置（Option Pattern）
// 用于在创建 HTTPServer 时注入自定义配置
type HTTPOption func(svr *HTTPServer) error

// StartHttpServer 启动一个 HTTP 服务
// 参数：
// - addr: 监听地址（如 ":8080"）
// - handler: HTTP 处理器
// - opts: 可选配置项
//
// 返回：
// - HTTPServer 实例
// - error
//
// 特性：
// 1. 内部创建 listener
// 2. 支持 context 控制生命周期
// 3. 启动 goroutine 异步运行服务
// 4. 自动处理关闭状态
func StartHttpServer(addr string, handler http.Handler, opts ...HTTPOption) (*HTTPServer, error) {
	// 创建 TCP 监听
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("listen errorr=", err)
		return nil, errors.New("Init listener fail")
	}

	// 创建服务级 context（用于整个 HTTP 生命周期）
	srvCtx, srvCancel := context.WithCancel(context.Background())

	// 初始化 http.Server
	srv := &http.Server{
		Handler:           handler,
		ReadTimeout:       timeouts.ReadTimeout,
		ReadHeaderTimeout: timeouts.ReadHeaderTimeout,
		WriteTimeout:      timeouts.WriteTimeout,
		IdleTimeout:       timeouts.IdleTimeout,

		// 为每个连接注入基础 context
		BaseContext: func(listener net.Listener) context.Context {
			return srvCtx
		},
	}

	out := &HTTPServer{
		listener: listener,
		srv:      srv,
	}

	// 应用自定义配置（Option Pattern）
	for _, opt := range opts {
		if err := opt(out); err != nil {
			// 配置失败，取消 context
			srvCancel()
			fmt.Println("apply err:", err)
			return nil, errors.New("One of http op fail")
		}
	}

	// 异步启动 HTTP 服务
	go func() {
		err := out.srv.Serve(listener)

		// 服务退出时取消 context
		srvCancel()

		// 判断是否是正常关闭
		if errors.Is(err, http.ErrServerClosed) {
			out.closed.Store(true)
		} else {
			// 非预期错误，直接 panic（说明是严重问题）
			fmt.Println("unknow err:", err)
			panic("unknow error")
		}
	}()

	return out, nil
}

// Closed 返回服务是否已经关闭
func (hs *HTTPServer) Closed() bool {
	return hs.closed.Load()
}

// Stop 优雅关闭服务（推荐使用）
//
// 行为：
// 1. 尝试优雅关闭（Shutdown）
// 2. 若超时（ctx 结束），则强制关闭（Close）
func (hs *HTTPServer) Stop(ctx context.Context) error {
	if err := hs.Shutdown(ctx); err != nil {
		// 如果是 context 超时导致 shutdown 失败
		if errors.Is(err, ctx.Err()) {
			// fallback：强制关闭
			return hs.Close()
		}
		return err
	}
	return nil
}

// Shutdown 优雅关闭 HTTP 服务
//
// 特点：
// - 等待已有连接处理完成
// - 不再接受新连接
func (hs *HTTPServer) Shutdown(ctx context.Context) error {
	return hs.srv.Shutdown(ctx)
}

// Close 强制关闭 HTTP 服务
//
// 特点：
// - 立即关闭所有连接（不保证请求完成）
// - 通常作为 Shutdown 的兜底方案
func (hs *HTTPServer) Close() error {
	return hs.srv.Close()
}

// Addr 返回监听地址（实际绑定地址）
//
// 常用于：
// - 获取随机端口（:0 场景）
// - 打印服务启动信息
func (hs *HTTPServer) Addr() net.Addr {
	return hs.listener.Addr()
}

// WithMaxHeaderBytes 设置 HTTP 请求头最大字节数
//
// 使用方式：
// StarHttpServer(":8080", handler, WithMaxHeaderBytes(1<<20))
func WithMaxHeaderBytes(max int) HTTPOption {
	return func(srv *HTTPServer) error {
		srv.srv.MaxHeaderBytes = max
		return nil
	}
}
