package retry

import (
	"context"
	"fmt"
	"time"
)

// ErrFailedPermanently 表示操作在达到最大重试次数后仍然失败的错误。
// 包含实际尝试次数和最后一次失败的原始错误。
type ErrFailedPermanently struct {
	attempts int
	LastErr  error
}

// Error 实现 error 接口，返回包含尝试次数和原始错误的描述字符串。
func (e *ErrFailedPermanently) Error() string {
	return fmt.Sprintf("operation failed permanently after %d attempts: %v", e.attempts, e.LastErr)
}

// Unwrap 返回最后一次失败的原始错误，支持 errors.Is / errors.As 链式解包。
func (e *ErrFailedPermanently) Unwrap() error {
	return e.LastErr
}

// pair 是一个泛型二元组，用于将两个返回值打包成单个值，
// 以便复用单返回值的 Do 函数。
type pair[T, U any] struct {
	a T
	b U
}

// Do2 是 Do 的双返回值变体，支持操作函数返回两个值加一个 error。
// 内部将两个返回值打包为 pair，委托给 Do 执行重试逻辑。
func Do2[T, U any](ctx context.Context, maxAttempts int, strategy Strategy, op func() (T, U, error)) (T, U, error) {
	f := func() (pair[T, U], error) {
		a, b, err := op()
		return pair[T, U]{a, b}, err
	}
	res, err := Do(ctx, maxAttempts, strategy, f)
	return res.a, res.b, err
}

// Do 以指定的重试策略执行操作，最多尝试 maxAttempts 次。
// 每次失败后根据 strategy 计算等待时长再重试（最后一次失败不等待）。
// 若 ctx 已取消则立即返回 ctx.Err()。
// 若所有尝试均失败，返回 *ErrFailedPermanently 包装最后一次错误。
func Do[T any](ctx context.Context, maxAttempts int, strategy Strategy, op func() (T, error)) (T, error) {
	var empty, ret T
	var err error
	if maxAttempts < 1 {
		return empty, fmt.Errorf("need at least 1 attempt to run op, but have %d max attempts", maxAttempts)
	}

	for i := 0; i < maxAttempts; i++ {
		// 每次尝试前检查上下文是否已取消
		if ctx.Err() != nil {
			return empty, ctx.Err()
		}
		ret, err = op()
		if err == nil {
			return ret, nil
		}
		// 最后一次失败不需要等待
		if i != maxAttempts-1 {
			time.Sleep(strategy.Duration(i))
		}
	}
	return empty, &ErrFailedPermanently{
		attempts: maxAttempts,
		LastErr:  err,
	}
}
