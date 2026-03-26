package cliapp

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/Sandwichzzy/market-services/common/opio"
)

// Lifecycle 定义应用生命周期接口，要求实现启动、停止和状态查询。
type Lifecycle interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Stopped() bool
}

// LifecycleAction 是创建 Lifecycle 实例的工厂函数类型。
// 接收 CLI 上下文和取消函数，返回已初始化的 Lifecycle 或错误。
type LifecycleAction func(ctx *cli.Context, close context.CancelCauseFunc) (Lifecycle, error)

// LifecycleCmd 将 LifecycleAction 包装为标准的 cli.ActionFunc，
// 使用默认的 opio.BlockOnInterruptsContext 监听系统中断信号。
func LifecycleCmd(fn LifecycleAction) cli.ActionFunc {
	return lifecycleCmd(fn, opio.BlockOnInterruptsContext)
}

// waitSignalFn 是阻塞等待信号的函数类型，便于测试时注入 mock 实现。
type waitSignalFn func(ctx context.Context, signals ...os.Signal)

// interruptErr 是收到中断信号时用于取消 context 的哨兵错误。
var interruptErr = errors.New("interrupt signal")

// lifecycleCmd 是 LifecycleCmd 的内部实现，接受可注入的信号等待函数，
// 便于单元测试时替换真实的信号监听逻辑。
//

// ---
// 执行流程
// LifecycleCmd(你的工厂函数)
// ↓
// 1. 创建 appCtx（带取消原因）
// 2. 后台监听 Ctrl+C 信号 → 收到后取消 appCtx
// 3. 调用工厂函数，初始化你的服务
// 4. 调用 Start() 启动服务
// 5. 阻塞等待 appCtx 取消（等信号或外部取消）
// 6. 创建新的 stopCtx，调用 Stop() 优雅停止
// └─ 停止阶段再次按 Ctrl+C → 强制退出
// ---
func lifecycleCmd(fn LifecycleAction, blockOnInterrupt waitSignalFn) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		hostCtx := ctx.Context
		appCtx, appCancel := context.WithCancelCause(hostCtx)
		ctx.Context = appCtx

		// 后台监听中断信号，收到后取消 appCtx
		go func() {
			blockOnInterrupt(appCtx)
			appCancel(interruptErr)
		}()

		appLifecycle, err := fn(ctx, appCancel)
		if err != nil {
			return errors.Join(
				fmt.Errorf("failed to setup: %w", err),
				context.Cause(appCtx),
			)
		}

		if err := appLifecycle.Start(appCtx); err != nil {
			return errors.Join(
				fmt.Errorf("failed to start: %w", err),
				context.Cause(appCtx),
			)
		}

		// 等待应用上下文取消（正常退出或收到信号）
		<-appCtx.Done()

		// 使用独立的 stopCtx 执行优雅停止，避免受 appCtx 取消影响
		stopCtx, stopCancel := context.WithCancelCause(hostCtx)
		go func() {
			// 在停止阶段再次监听中断信号，支持强制退出
			blockOnInterrupt(stopCtx)
			stopCancel(interruptErr)
		}()

		stopErr := appLifecycle.Stop(stopCtx)
		stopCancel(nil)
		if stopErr != nil {
			return errors.Join(
				fmt.Errorf("failed to stop: %w", stopErr),
				context.Cause(stopCtx),
			)
		}
		return nil
	}
}
