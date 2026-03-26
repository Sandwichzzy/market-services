package main

import (
	"context"
	"fmt"

	"github.com/Sandwichzzy/market-services/common/opio"
	"github.com/Sandwichzzy/market-services/config"
	"github.com/Sandwichzzy/market-services/database"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/Sandwichzzy/market-services/common/cliapp"
	flags2 "github.com/Sandwichzzy/market-services/flags"
)

// runRpc 启动 gRPC 服务，实现 cliapp.LifecycleAction 签名。
// shutdown 可用于在服务内部主动触发优雅退出。
// 当前为占位实现，待后续补充具体逻辑。
func runRpc(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	fmt.Println("running grpc services...")
	return nil, nil
}

// runMigrations 执行数据库迁移命令。
// 流程：注册中断信号取消 → 读取配置 → 连接数据库 → 执行迁移文件 → 关闭连接。
// 数据库连接通过 defer 确保在函数退出时关闭，关闭失败仅记录日志不影响返回值。
func runMigrations(ctx *cli.Context) error {
	// 将中断信号绑定到 ctx，收到 SIGINT/SIGTERM 时自动取消
	ctx.Context = opio.CancelOnInterrupt(ctx.Context)
	log.Info("running migrations...")
	cfg := config.NewConfig(ctx)
	db, err := database.NewDB(ctx.Context, cfg.MasterDB)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		return err
	}
	defer func(db *database.DB) {
		err := db.Close()
		if err != nil {
			log.Error("fail to close database", "err", err)
		}
	}(db)
	return db.ExecuteSQLMigration(cfg.Migrations)
}

// runRestApi 启动 REST API 服务，实现 cliapp.LifecycleAction 签名。
// 当前为占位实现，待后续补充具体逻辑。
func runRestApi(ctx *cli.Context, shutdown context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	fmt.Println("running rest api...")
	return nil, nil
}

// NewCli 构建并返回应用的 CLI 入口。
// GitCommit 和 GitData 预留用于版本信息展示（当前未使用）。
// 注册以下子命令：
//   - migrate：执行数据库迁移
//   - version：打印版本号
func NewCli(GitCommit string, GitData string) *cli.App {
	flags := flags2.Flags
	return &cli.App{
		Version:              "0.0.1",
		Description:          "An market services with rpc",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "migrate",
				Flags:       flags,
				Description: "Run database migrations",
				Action:      runMigrations,
			},
			{
				Name:        "version",
				Description: "Show project version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}
