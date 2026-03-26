package retry

import (
	"math"
	"math/rand"
	"time"
)

// Strategy 定义重试等待策略接口。
// Duration 根据当前尝试次数（从 0 开始）返回下次重试前的等待时长。
type Strategy interface {
	Duration(attempt int) time.Duration
}

// ExponentialStrategy 实现指数退避策略，等待时长随尝试次数指数增长，
// 并可叠加随机抖动（jitter）以避免多个客户端同时重试造成的惊群效应。
type ExponentialStrategy struct {
	Min       time.Duration // 最小等待时长（基础值）
	Max       time.Duration // 最大等待时长上限
	MaxJitter time.Duration // 随机抖动的最大值，0 表示不添加抖动
}

// Duration 计算第 attempt 次重试的等待时长。
// 公式：Min + 2^attempt 秒 + [0, MaxJitter) 随机抖动，结果不超过 Max。
// 当 attempt < 0 时直接返回 Min + jitter。
func (e *ExponentialStrategy) Duration(attempt int) time.Duration {
	var jitter time.Duration // 非负随机抖动
	if e.MaxJitter > 0 {
		jitter = time.Duration(rand.Int63n(e.MaxJitter.Nanoseconds()))
	}
	if attempt < 0 {
		return e.Min + jitter
	}
	durFloat := float64(e.Min)
	durFloat += math.Pow(2, float64(attempt)) * float64(time.Second)
	dur := time.Duration(durFloat)
	if durFloat > float64(e.Max) {
		dur = e.Max
	}
	dur += jitter

	return dur
}

// Exponential 返回默认配置的指数退避策略：
// 最小等待 0，最大等待 10s，最大抖动 250ms。
func Exponential() Strategy {
	return &ExponentialStrategy{
		Min:       0,
		Max:       10 * time.Second,
		MaxJitter: 250 * time.Millisecond,
	}
}

// FixedStrategy 实现固定间隔重试策略，每次等待相同的时长。
type FixedStrategy struct {
	Dur time.Duration // 固定等待时长
}

// Duration 始终返回固定的等待时长，忽略 attempt 参数。
func (f *FixedStrategy) Duration(attempt int) time.Duration {
	return f.Dur
}

// Fixed 返回使用指定固定时长的重试策略。
func Fixed(dur time.Duration) Strategy {
	return &FixedStrategy{
		Dur: dur,
	}
}
