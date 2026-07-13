package flags

import "github.com/urfave/cli/v2"

// 外部包（通常是 main 包）可以导入这个 flags 包，并将 Flags 添加到 cli.App 的标志列表中，从而完成配置注入。
const evnVarPrefix = "MARKET"

func prefixEnvVars(name string) []string {
	return []string{evnVarPrefix + "_" + name}
}

// 优先级（命令行参数 > 环境变量 > 默认值）
var (
	MigrationsFlag = &cli.StringFlag{
		Name:    "migrations-dir",                //命令行标志的名称
		Value:   "./migrations",                  //默认值
		Usage:   "path for database migrations",  //帮助信息中显示的说明
		EnvVars: prefixEnvVars("MIGRATIONS_DIR"), //对应的环境变量列表
	}

	// RpcHostFlag RPC Service
	RpcHostFlag = &cli.StringFlag{
		Name:     "rpc-host",
		Usage:    "The host of the rpc",
		EnvVars:  prefixEnvVars("RPC_HOST"),
		Required: true, //表示该标志必须在命令行或环境变量中提供
	}

	RpcPortFlag = &cli.IntFlag{
		Name:     "rpc-port",
		Usage:    "The port of the rpc",
		EnvVars:  prefixEnvVars("RPC_PORT"),
		Required: true,
	}

	// HttpHostFlag RPC Service
	HttpHostFlag = &cli.StringFlag{
		Name:     "http-host",
		Usage:    "The host of the http",
		EnvVars:  prefixEnvVars("HTTP_HOST"),
		Required: true,
	}
	HttpPortFlag = &cli.IntFlag{
		Name:     "http-port",
		Usage:    "The port of the http",
		EnvVars:  prefixEnvVars("HTTP_PORT"),
		Required: true,
	}

	// MasterDbHostFlag Flags
	MasterDbHostFlag = &cli.StringFlag{
		Name:     "master-db-host",
		Usage:    "The host of the master database",
		EnvVars:  prefixEnvVars("MASTER_DB_HOST"),
		Required: true,
	}
	MasterDbPortFlag = &cli.IntFlag{
		Name:     "master-db-port",
		Usage:    "The port of the master database",
		EnvVars:  prefixEnvVars("MASTER_DB_PORT"),
		Required: true,
	}
	MasterDbUserFlag = &cli.StringFlag{
		Name:     "master-db-user",
		Usage:    "The user of the master database",
		EnvVars:  prefixEnvVars("MASTER_DB_USER"),
		Required: true,
	}
	MasterDbPasswordFlag = &cli.StringFlag{
		Name:     "master-db-password",
		Usage:    "The host of the master database",
		EnvVars:  prefixEnvVars("MASTER_DB_PASSWORD"),
		Required: true,
	}
	MasterDbNameFlag = &cli.StringFlag{
		Name:     "master-db-name",
		Usage:    "The db name of the master database",
		EnvVars:  prefixEnvVars("MASTER_DB_NAME"),
		Required: true,
	}

	// Slave DB  flags
	SlaveDbHostFlag = &cli.StringFlag{
		Name:    "slave-db-host",
		Usage:   "The host of the slave database",
		EnvVars: prefixEnvVars("SLAVE_DB_HOST"),
	}
	SlaveDbPortFlag = &cli.IntFlag{
		Name:    "slave-db-port",
		Usage:   "The port of the slave database",
		EnvVars: prefixEnvVars("SLAVE_DB_PORT"),
	}
	SlaveDbUserFlag = &cli.StringFlag{
		Name:    "slave-db-user",
		Usage:   "The user of the slave database",
		EnvVars: prefixEnvVars("SLAVE_DB_USER"),
	}
	SlaveDbPasswordFlag = &cli.StringFlag{
		Name:    "slave-db-password",
		Usage:   "The host of the slave database",
		EnvVars: prefixEnvVars("SLAVE_DB_PASSWORD"),
	}
	SlaveDbNameFlag = &cli.StringFlag{
		Name:    "slave-db-name",
		Usage:   "The db name of the slave database",
		EnvVars: prefixEnvVars("SLAVE_DB_NAME"),
	}

	MetricsHostFlag = &cli.StringFlag{
		Name:     "metric-host",
		Usage:    "The host of the metric",
		EnvVars:  prefixEnvVars("METRIC_HOST"),
		Required: true,
	}
	MetricsPortFlag = &cli.IntFlag{
		Name:     "metric-port",
		Usage:    "The port of the metric",
		EnvVars:  prefixEnvVars("METRIC_PORT"),
		Required: true,
	}

	RedisAddressFlag = &cli.StringFlag{
		Name:     "redis-address",
		Usage:    "The address of the redis",
		EnvVars:  prefixEnvVars("REDIS_ADDRESS"),
		Required: true,
	}
	RedisPasswordFlag = &cli.StringFlag{
		Name:     "redis-password",
		Usage:    "The password of the redis",
		EnvVars:  prefixEnvVars("REDIS_PASSWORD"),
		Required: true,
	}
	RedisDbIndexFlag = &cli.IntFlag{
		Name:     "redis-db-index",
		Usage:    "The DB index of the redis",
		EnvVars:  prefixEnvVars("REDIS_DB_INDEX"),
		Required: true,
	}

	CrawlerProxyFlag = &cli.StringFlag{
		Name:    "crawler-proxy",
		Usage:   "Optional proxy address for exchange crawler requests, e.g. http://127.0.0.1:7890",
		EnvVars: prefixEnvVars("CRAWLER_PROXY"),
	}
	CrawlerProxyTypeFlag = &cli.StringFlag{
		Name:    "crawler-proxy-type",
		Usage:   "Optional proxy type for exchange crawler requests: http or socks5",
		Value:   "http",
		EnvVars: prefixEnvVars("CRAWLER_PROXY_TYPE"),
	}
	ChainRpcUrlFlag = &cli.StringFlag{
		Name:     "chain-rpc-url",
		Usage:    " chain rpc url",
		EnvVars:  prefixEnvVars("CHAIN_RPC_URL"),
		Required: true,
	}
	RouteAddressFlag = &cli.StringFlag{
		Name:     "route-address",
		Usage:    "dex router address",
		EnvVars:  prefixEnvVars("ROUTE_ADDRESS"),
		Required: true,
	}
	QuoteTokenAddressFlag = &cli.StringFlag{
		Name:     "quote-token-address",
		Usage:    "dex quote token address used for price direction",
		EnvVars:  prefixEnvVars("QUOTE_TOKEN_ADDRESS"),
		Required: true,
	}
	TokenPairFlag = &cli.StringFlag{
		Name:     "token-pair",
		Usage:    "token pair",
		EnvVars:  prefixEnvVars("TOKEN_PAIR"),
		Required: true,
	}
)

var requireFlags = []cli.Flag{
	MigrationsFlag,
	RpcHostFlag,
	RpcPortFlag,
	HttpHostFlag,
	HttpPortFlag,
	MasterDbHostFlag,
	MasterDbPortFlag,
	MasterDbUserFlag,
	MasterDbPasswordFlag,
	MasterDbNameFlag,
	RedisAddressFlag,
	RedisPasswordFlag,
	RedisDbIndexFlag,
	ChainRpcUrlFlag,
	RouteAddressFlag,
	QuoteTokenAddressFlag,
	TokenPairFlag,
}

var optionalFlags = []cli.Flag{
	SlaveDbHostFlag,
	SlaveDbPortFlag,
	SlaveDbUserFlag,
	SlaveDbPasswordFlag,
	SlaveDbNameFlag,
	MetricsHostFlag,
	MetricsPortFlag,
	CrawlerProxyFlag,
	CrawlerProxyTypeFlag,
}

func init() {
	Flags = append(requireFlags, optionalFlags...) //将两组标志合并到导出的变量 Flags 中
}

var Flags []cli.Flag
