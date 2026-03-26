package cliapp

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// CloneableGeneric 扩展了 cli.Generic 接口，要求实现 Clone 方法。
// 用于支持对 GenericFlag 值的深拷贝，避免多个命令共享同一个 flag 实例时的状态污染。
type CloneableGeneric interface {
	cli.Generic
	Clone() any
}

// ProtectFlags 对传入的 flag 列表进行深拷贝，返回独立的副本切片。
// 主要用于保护全局 flag 定义不被子命令的解析过程修改。
// 若某个 GenericFlag 的值未实现 CloneableGeneric，则会 panic。
func ProtectFlags(flags []cli.Flag) []cli.Flag {
	out := make([]cli.Flag, 0, len(flags))
	for _, f := range flags {
		fCopy, err := cloneFlag(f)
		if err != nil {
			panic(fmt.Errorf("failed to clone flag %q: %w", f.Names()[0], err))
		}
		out = append(out, fCopy)
	}
	return out
}

// cloneFlag 对单个 cli.Flag 进行克隆。
// 对于 *cli.GenericFlag：要求其 Value 实现 CloneableGeneric，并对 Value 进行深拷贝。
// 对于其他类型的 flag：直接返回原始引用（值类型 flag 无需深拷贝）。
func cloneFlag(f cli.Flag) (cli.Flag, error) {
	switch typedFlag := f.(type) {
	case *cli.GenericFlag:
		if genValue, ok := typedFlag.Value.(CloneableGeneric); ok {
			cpy := *typedFlag // 浅拷贝 flag 结构体
			cpyVal, ok := genValue.Clone().(cli.Generic)
			if !ok {
				return nil, fmt.Errorf("cloned Generic value is not Generic: %T", typedFlag)
			}
			cpy.Value = cpyVal
			return &cpy, nil
		} else {
			return nil, fmt.Errorf("cannot clone Generic value: %T", typedFlag)
		}
	default:
		// 非 GenericFlag 类型无需深拷贝，直接复用
		return f, nil
	}
}
