# market-services

`market-services` 是一个 Go 编写的行情服务单仓库，包含行情抓取、法币汇率入库、聚合行情计算、REST API、gRPC API 和后台 Worker。

## 功能概览

- **行情数据**：从交易所抓取交易对价格、盘口和 K 线数据。
- **法币汇率**：从 ExchangeRate-API 等平台拉取法币汇率并入库。
- **聚合处理**：由 Worker 对跨交易所行情进行聚合，生成 `symbol_market` 和 `symbol_market_currency` 数据。
- **REST API**：对外提供资产、交易对、行情、法币、K 线查询接口，详见 [`REST_API.md`](REST_API.md)。
- **gRPC API**：通过 `proto/market_services.proto` 定义对外 RPC 接口，生成代码位于 `services/grpc/proto/`。
- **数据库迁移**：SQL 迁移文件位于 `migrations/`。

## 项目结构

```text
cmd/market-services/      CLI 入口，包含 migrate/rpc/api/crawler/worker/version 子命令
common/                   生命周期、HTTP server、重试、信号处理等公共工具
config/                   配置装配与 API key YAML 配置
crawler/                  交易所行情与法币汇率抓取逻辑
database/                 GORM 数据模型与仓储层
flags/                    CLI flags 与 MARKET_ 环境变量定义
migrations/               PostgreSQL 数据库迁移脚本
proto/                    protobuf 源文件
redis/                    Redis 客户端封装
services/grpc/            gRPC 服务、handler 与生成代码
services/http/            REST API 路由、模型和服务层
worker/                   后台聚合任务
```

## 环境要求

- Go `1.25.1` 或与 `go.mod` 兼容的版本
- PostgreSQL
- Redis
- `protoc`，仅在需要重新生成 protobuf 代码时需要
- `protoc-gen-go`、`protoc-gen-go-grpc`，可通过 `make proto` 自动安装

数据库迁移会创建：

- `uuid-ossp` 扩展
- `UINT256` 自定义 domain
- `asset`、`exchange`、`symbol`、`currency`、`exchange_symbol`、`symbol_market`、`symbol_market_currency`、K 线等业务表

## 配置

配置来源优先级：

```text
CLI 参数 > MARKET_ 环境变量 > 默认值
```

常用环境变量：

```bash
export MARKET_RPC_HOST=0.0.0.0
export MARKET_RPC_PORT=9090

export MARKET_HTTP_HOST=0.0.0.0
export MARKET_HTTP_PORT=8080

export MARKET_MASTER_DB_HOST=127.0.0.1
export MARKET_MASTER_DB_PORT=5432
export MARKET_MASTER_DB_USER=postgres
export MARKET_MASTER_DB_PASSWORD=postgres
export MARKET_MASTER_DB_NAME=market_services

export MARKET_REDIS_ADDRESS=127.0.0.1:6379
export MARKET_REDIS_PASSWORD=
export MARKET_REDIS_DB_INDEX=0

export MARKET_MIGRATIONS_DIR=./migrations
```

API Key 配置文件默认读取：

```text
config/api-keys.yaml
```

可从示例文件复制：

```bash
cp config/api-keys.yaml.example config/api-keys.yaml
```

示例结构：

```yaml
api_keys:
  exchange_rate: "your-exchangerate-api-key-here"
  coin_market_cap: "your-coinmarketcap-api-key-here"
  fixer_io: "your-fixer-api-key-here"
  open_exchange_rates: "your-openexchangerates-api-key-here"
  currency: "your-currencyapi-key-here"
  currency_beacon: "your-currencybeacon-api-key-here"
  currency_freaks: "your-currencyfreaks-api-key-here"
```

## 构建与测试

构建二进制：

```bash
make market-services
```

运行测试：

```bash
make test
```

运行 lint：

```bash
make lint
```

重新生成 protobuf 代码：

```bash
make proto
```

清理构建产物：

```bash
make clean
```

## 运行

执行数据库迁移：

```bash
./market-services migrate
```

启动 REST API：

```bash
./market-services api
```

启动 gRPC 服务：

```bash
./market-services rpc
```

启动爬虫服务：

```bash
./market-services crawler
```

启动 Worker：

```bash
./market-services worker
```

查看版本：

```bash
./market-services version
```

## REST API

业务 REST API 当前统一使用：

```text
POST + application/json request body
```

健康检查：

```http
GET /healthz
```

已实现接口包括：

- `POST /api/v1/get_support_assets`
- `POST /api/v1/list_market_symbols`
- `POST /api/v1/list_symbol_markets`
- `POST /api/v1/get_symbol_market`
- `POST /api/v1/list_currencies`
- `POST /api/v1/get_symbol_market_currency`
- `POST /api/v1/list_klines`
- `POST /api/v1/list_exchange_klines`

完整请求与响应示例见 [`REST_API.md`](REST_API.md)。

## gRPC API

protobuf 定义位于：

```text
proto/market_services.proto
```

生成代码位于：

```text
services/grpc/proto/
```

当前服务分组：

- `MarketServices`：资产相关接口
- `MarketSymbolService`：交易对与聚合行情接口
- `FiatCurrencyService`：法币相关接口
- `KlineService`：聚合 K 线与交易所维度 K 线接口

重新生成：

```bash
make proto
```

## 数据流概览

```text
crawler
  -> exchange_symbol / exchange_symbol_kline / currency
worker
  -> symbol_market / symbol_market_currency / symbol_kline
services/http, services/grpc
  -> 对外查询接口
```

说明：

- `exchange_symbol` 存储交易所维度的最新行情。
- `symbol_market` 存储全市场聚合行情。
- `symbol_market_currency` 存储交易对按法币换算后的最新行情，当前按 `(symbol_guid, currency_guid)` 唯一。
- `exchange_symbol_kline` 存储交易所维度 K 线。
- `symbol_kline` 存储聚合后的交易对 K 线。
- 当前 K 线对外接口只暴露 `1m` 周期。

## 开发注意事项

- 不要手写修改 `services/grpc/proto/` 下的生成代码；修改 `proto/*.proto` 后执行 `make proto`。
- 价格、成交量、市值等高精度数值在 API 中以字符串返回，避免浮点精度损失。
- `MARKET_` 环境变量由 `flags/flags.go` 统一定义。
- 数据库 schema 变更应新增 SQL migration，并随代码一同提交。
- 业务接口返回码约定详见 [`REST_API.md`](REST_API.md)。

