package config

import (
	"github.com/Sandwichzzy/market-services/flags"
	"github.com/urfave/cli/v2"
)

// 将从命令行（或环境变量）解析到的参数，转换为结构化的配置对象，供应用程序其他部分直接使用。
// 业务代码无需关心配置来源（命令行参数、环境变量），只需读取 Config 对象，降低了耦合度。
// cli.Context 提供的 String、Int 等方法返回具体类型，避免了手动类型转换错误。
// 所有配置项的定义在 flags 包，组装逻辑在 config 包，便于维护和扩展。新增配置只需修改 flags 和 config 两处。
type Config struct {
	Migrations            string
	RpcServer             ServerConfig
	RestServer            ServerConfig
	RedisConfig           RedisConfig
	Metrics               ServerConfig
	MasterDB              DBConfig
	SlaveDB               DBConfig
	ExchangeRatePlatforms []ExchangeRatePlatformConfig
	BaseCurrency          string
	APIKeyConfig          APIKeyConfig
}

type APIKeyConfig struct {
	ExchangeRate      string `yaml:"exchange_rate"`
	FixerIO           string `yaml:"fixer_io"`
	OpenExchangeRates string `yaml:"open_exchange_rates"`
	Currency          string `yaml:"currency"`
	CurrencyBeacon    string `yaml:"currency_beacon"`
	CurrencyFreaks    string `yaml:"currency_freaks"`
}
type ServerConfig struct {
	Host string
	Port int
}
type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`     // Redis地址，格式: host:port
	Password string `yaml:"password"` // Redis密码（可选）
	DB       int    `yaml:"db"`       // Redis数据库索引
}

type ExchangeRatePlatformConfig struct {
	Name    string
	BaseURL string
}

func NewConfig(ctx *cli.Context) Config {
	return Config{
		Migrations: ctx.String(flags.MigrationsFlag.Name),
		RpcServer: ServerConfig{
			Host: ctx.String(flags.RpcHostFlag.Name),
			Port: ctx.Int(flags.RpcPortFlag.Name),
		},
		RestServer: ServerConfig{
			Host: ctx.String(flags.HttpHostFlag.Name),
			Port: ctx.Int(flags.HttpPortFlag.Name),
		},
		Metrics: ServerConfig{
			Host: ctx.String(flags.MetricsHostFlag.Name),
			Port: ctx.Int(flags.MetricsPortFlag.Name),
		},
		MasterDB: DBConfig{
			Host:     ctx.String(flags.MasterDbHostFlag.Name),
			Port:     ctx.Int(flags.MasterDbPortFlag.Name),
			Name:     ctx.String(flags.MasterDbNameFlag.Name),
			User:     ctx.String(flags.MasterDbUserFlag.Name),
			Password: ctx.String(flags.MasterDbPasswordFlag.Name),
		},
		SlaveDB: DBConfig{
			Host:     ctx.String(flags.SlaveDbHostFlag.Name),
			Port:     ctx.Int(flags.SlaveDbPortFlag.Name),
			Name:     ctx.String(flags.SlaveDbNameFlag.Name),
			User:     ctx.String(flags.SlaveDbUserFlag.Name),
			Password: ctx.String(flags.SlaveDbPasswordFlag.Name),
		},
		RedisConfig: RedisConfig{
			Addr:     ctx.String(flags.RedisAddressFlag.Name),
			Password: ctx.String(flags.RedisPasswordFlag.Name),
			DB:       ctx.Int(flags.RedisDbIndexFlag.Name),
		},
	}
}
