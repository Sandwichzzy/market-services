package tasks

import (
	"fmt"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

// Group is a tasks group, which can at any point be awaited to complete.
// Tasks in the group are run in separate go routines.
// If a task panics, the panic is recovered with HandleCrit.
type Group struct {
	errGroup   errgroup.Group
	HandleCrit func(err error)
}

// t.errGroup.Go 是 golang.org/x/sync/errgroup 包中 Group 类型的方法，用于启动一个并发任务，并在任务返回错误时记录下来，
func (t *Group) Go(fn func() error) {
	t.errGroup.Go(func() error {
		//Go 方法内部使用 defer recover() 捕获 panic，避免未处理的 panic 使整个进程退出。
		defer func() {
			if err := recover(); err != nil {
				//捕获 panic 后调用 debug.PrintStack() 输出堆栈信息，方便定位问题根源。
				debug.PrintStack()
				//通过 HandleCrit 回调将 panic 转换为普通 error 并交给调用方自定义处理（例如记录日志、发送告警、优雅关闭等），使错误处理逻辑可配置。
				t.HandleCrit(fmt.Errorf("panic: %v", err))
			}
		}()
		return fn()
	})
}

// Wait 会等待所有任务完成并返回第一个非 nil 错误（或 nil）。
func (t *Group) Wait() error {
	return t.errGroup.Wait()
}
