package opio

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// DefaultInterruptSignals 是默认监听的系统中断信号集合，
// 包括 Ctrl+C (SIGINT)、SIGKILL、SIGTERM 和 SIGQUIT。
var DefaultInterruptSignals = []os.Signal{
	os.Interrupt,
	os.Kill,
	syscall.SIGTERM,
	syscall.SIGQUIT,
}

// BlockOnInterrupts 阻塞当前 goroutine，直到收到指定的系统信号。
// 若未传入 signals，则使用 DefaultInterruptSignals。
func BlockOnInterrupts(signals ...os.Signal) {
	if len(signals) == 0 {
		signals = DefaultInterruptSignals
	}
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, signals...)
	<-interruptChannel
}

// BlockOnInterruptsContext 阻塞当前 goroutine，直到收到指定信号或 ctx 被取消。
// 当 ctx 取消时会调用 signal.Stop 停止信号通知，避免 goroutine 泄漏。
func BlockOnInterruptsContext(ctx context.Context, signals ...os.Signal) {
	if len(signals) == 0 {
		signals = DefaultInterruptSignals
	}
	//创建了一个容量为 1 的缓冲 channel，用于接收操作系统信号。
	interruptChannel := make(chan os.Signal, 1)
	//将指定的信号（或默认中断信号）转发到这个 channel。
	signal.Notify(interruptChannel, signals...)
	select {
	//当操作系统发送了匹配的信号时，该 case 会收到信号并立即执行，函数返回。
	case <-interruptChannel:
	case <-ctx.Done():
		signal.Stop(interruptChannel) //停止信号通知，避免 goroutine 泄漏，然后函数返回
	}
}

// interruptContextKeyType 是存储在 context 中的中断阻塞函数的键类型，
// 使用私有结构体类型避免与其他包的 key 冲突。
type interruptContextKeyType struct{}

// blockerContextKey 是在 context 中存取 BlockFn 的键。
var blockerContextKey = interruptContextKeyType{}

// interruptCatcher 持有一个信号接收通道，用于捕获系统中断信号。
type interruptCatcher struct {
	incoming chan os.Signal
}

// Block 阻塞直到收到信号或 ctx 被取消。
func (c *interruptCatcher) Block(ctx context.Context) {
	select {
	case <-c.incoming:
	case <-ctx.Done():
	}
}

// WithInterruptBlocker 向 ctx 中注入一个中断捕获器（BlockFn）。
// 若 ctx 中已存在中断处理器则直接返回原 ctx，避免重复注册。
func WithInterruptBlocker(ctx context.Context) context.Context {
	if ctx.Value(blockerContextKey) != nil { // 已有中断处理器，直接返回
		return ctx
	}
	catcher := &interruptCatcher{
		incoming: make(chan os.Signal, 10),
	}
	signal.Notify(catcher.incoming, DefaultInterruptSignals...)

	return context.WithValue(ctx, blockerContextKey, BlockFn(catcher.Block))
}

// WithBlocker 向 ctx 中注入自定义的 BlockFn，用于测试或替换默认信号行为。
func WithBlocker(ctx context.Context, fn BlockFn) context.Context {
	return context.WithValue(ctx, blockerContextKey, fn)
}

// BlockFn 是一个阻塞函数类型，接收 ctx 并在信号到来或 ctx 取消时返回。
type BlockFn func(ctx context.Context)

// BlockerFromContext 从 ctx 中取出注入的 BlockFn。
// 若未注入则返回 nil，调用方需自行处理 nil 情况。
func BlockerFromContext(ctx context.Context) BlockFn {
	v := ctx.Value(blockerContextKey)
	if v == nil {
		return nil
	}
	return v.(BlockFn)
}

// CancelOnInterrupt 返回一个新的 context，当收到中断信号时自动取消。
// 优先使用 ctx 中注入的 BlockFn；若未注入则使用默认信号监听。
func CancelOnInterrupt(ctx context.Context) context.Context {

	inner, cancel := context.WithCancel(ctx)

	blockOnInterrupt := BlockerFromContext(ctx)
	if blockOnInterrupt == nil {
		// 未注入自定义阻塞函数，使用默认信号监听
		blockOnInterrupt = func(ctx context.Context) {
			BlockOnInterruptsContext(ctx) // default signals
		}
	}

	go func() {
		blockOnInterrupt(ctx)
		cancel()
	}()

	return inner
}
