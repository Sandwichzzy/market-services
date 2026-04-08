package config

import (
	"fmt"
	"os"

	"github.com/Sandwichzzy/market-services/flags"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

const (
	defaultAPIKeysConfigPath      = "config/api-keys.yaml"
	defaultBaseCurrency           = "USD"
	defaultExchangeRateAPIBaseURL = "https://v6.exchangerate-api.com/v6"
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

// 存储各平台的API认证密钥：每个字段对应一个平台的API密钥
// 用于API请求认证：在 fiatcurrency.go:25-33 中，将密钥映射到平台名称传递给 ExchangeRateWorker
// 安全管理：通过环境变量注入，避免硬编码敏感信息
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

// (平台配置) 决定启用哪些汇率平台：配置项是一个数组，包含你想使用的所有平台
// 指定平台的访问地址：每个平台可以自定义BaseURL（用于自建服务或代理）
// 构建请求策略：在 fiatcurrency_client.go:329 的 BuildStrategyConfigs()
type ExchangeRatePlatformConfig struct {
	Name    string
	BaseURL string
}

type apiKeyFileConfig struct {
	APIKeys *APIKeyConfig `yaml:"api_keys"`
}

func loadAPIKeyConfig(path string) (APIKeyConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return APIKeyConfig{}, fmt.Errorf("read %s: %w", path, err)
	}

	var parsed apiKeyFileConfig
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return APIKeyConfig{}, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	if parsed.APIKeys == nil {
		return APIKeyConfig{}, fmt.Errorf("missing api_keys section in %s", path)
	}

	return *parsed.APIKeys, nil
}

func defaultExchangeRatePlatforms() []ExchangeRatePlatformConfig {
	return []ExchangeRatePlatformConfig{
		{
			Name:    "ExchangeRate-API",
			BaseURL: defaultExchangeRateAPIBaseURL,
		},
	}
}

func NewConfig(ctx *cli.Context) (Config, error) {
	apiKeyConfig, err := loadAPIKeyConfig(defaultAPIKeysConfigPath)
	if err != nil {
		return Config{}, fmt.Errorf("load api key config: %w", err)
	}

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
		ExchangeRatePlatforms: defaultExchangeRatePlatforms(),
		BaseCurrency:          defaultBaseCurrency,
		APIKeyConfig:          apiKeyConfig,
	}, nil
}
