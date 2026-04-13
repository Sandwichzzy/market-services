# Market Services REST API

本文档描述当前 `market-services` 已实现的对外 REST API。

## 基本说明

- Base Path：`/api/v1`
- 请求方式：当前业务接口统一使用 `POST`+ ` JSON Body ` 复杂查询
- 请求体格式：`application/json`
- 健康检查：`GET /healthz`
- 响应结构：各接口统一返回 `code`、`message`，列表接口额外返回 `pagination`

## 返回码约定

- `200`：成功
- `400`：请求参数错误
- `404`：数据不存在
- `4000`：服务内部错误

## 分页对象

请求分页：

```json
{
  "pagination": {
    "page": 1,
    "page_size": 10
  }
}
```

响应分页：

```json
{
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total": 120
  }
}
```

默认值：

- `page` 默认 `1`
- `page_size` 默认 `10`

## 1. 支持资产列表

### `POST /api/v1/get_support_assets`

查询支持的资产列表。

请求示例：

```json
{
  "consumerToken": "demo-token"
}
```

响应示例：

```json
{
  "code": 200,
  "message": "here is support asset list, your query token=demo-token",
  "result": [
    {
      "guid": "asset-guid",
      "asset_name": "Bitcoin",
      "asset_symbol": "BTC",
      "asset_logo": "https://example.com/btc.png"
    }
  ]
}
```

## 2. 交易对基础信息

### `POST /api/v1/list_market_symbols`

查询交易对基础信息。

请求字段：

- `pagination.page`
- `pagination.page_size`
- `only_active`
- `base_asset_guid`
- `quote_asset_guid`
- `market_type`

请求示例：

```json
{
  "pagination": {
    "page": 1,
    "page_size": 20
  },
  "only_active": true,
  "base_asset_guid": "",
  "quote_asset_guid": "",
  "market_type": "SPOT"
}
```

响应示例：

```json
{
  "code": 200,
  "message": "list market symbols success",
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 1
  },
  "result": [
    {
      "guid": "symbol-guid",
      "symbol_name": "BTC/USDT",
      "base_asset_guid": "btc-guid",
      "quote_asset_guid": "usdt-guid",
      "market_type": "SPOT",
      "is_active": true,
      "created_at": 1712900000000,
      "updated_at": 1712900000000
    }
  ]
}
```

说明：

- 当前库内 `market_type` 实际主要是 `SPOT`

## 3. 聚合行情

### `POST /api/v1/list_symbol_markets`

分页查询聚合行情快照。

请求字段：

- `pagination.page`
- `pagination.page_size`
- `only_active`
- `symbol_guid`

请求示例：

```json
{
  "pagination": {
    "page": 1,
    "page_size": 20
  },
  "only_active": true,
  "symbol_guid": ""
}
```

响应示例：

```json
{
  "code": 200,
  "message": "list symbol markets success",
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 1
  },
  "result": [
    {
      "guid": "market-guid",
      "symbol_guid": "symbol-guid",
      "price": "65000.12",
      "ask_price": "65000.50",
      "bid_price": "64999.90",
      "volume": "123456789",
      "market_cap": "456789123",
      "radio": "0.0123",
      "is_active": true,
      "created_at": 1712900000000,
      "updated_at": 1712900000000
    }
  ]
}
```

### `POST /api/v1/get_symbol_market`

查询单个交易对最新聚合行情。

请求示例：

```json
{
  "symbol_guid": "symbol-guid"
}
```

响应示例：

```json
{
  "code": 200,
  "message": "get symbol market success",
  "result": {
    "guid": "market-guid",
    "symbol_guid": "symbol-guid",
    "price": "65000.12",
    "ask_price": "65000.50",
    "bid_price": "64999.90",
    "volume": "123456789",
    "market_cap": "456789123",
    "radio": "0.0123",
    "is_active": true,
    "created_at": 1712900000000,
    "updated_at": 1712900000000
  }
}
```

错误示例：

```json
{
  "code": 400,
  "message": "symbol_guid is required"
}
```

## 4. 法币业务

### `POST /api/v1/list_currencies`

分页查询法币配置与汇率。

请求字段：

- `pagination.page`
- `pagination.page_size`
- `only_active`
- `currency_code`

请求示例：

```json
{
  "pagination": {
    "page": 1,
    "page_size": 20
  },
  "only_active": true,
  "currency_code": "USD"
}
```

响应示例：

```json
{
  "code": 200,
  "message": "list currencies success",
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 1
  },
  "result": [
    {
      "guid": "currency-guid",
      "currency_name": "US Dollar",
      "currency_code": "USD",
      "rate": "1",
      "buy_spread": "0",
      "sell_spread": "0",
      "is_active": true,
      "created_at": 1712900000000,
      "updated_at": 1712900000000
    }
  ]
}
```

### `POST /api/v1/get_symbol_market_currency`

按唯一键 `symbol_guid + currency_guid` 查询单条法币计价行情。

请求示例：

```json
{
  "symbol_guid": "symbol-guid",
  "currency_guid": "currency-guid"
}
```

响应示例：

```json
{
  "code": 200,
  "message": "get symbol market currency success",
  "result": {
    "guid": "smc-guid",
    "symbol_guid": "symbol-guid",
    "currency_guid": "currency-guid",
    "price": "470000.12",
    "ask_price": "470010.22",
    "bid_price": "469990.11",
    "is_active": true,
    "created_at": 1712900000000,
    "updated_at": 1712900000000
  }
}
```

说明：

- `symbol_market_currency` 当前按 `(symbol_guid, currency_guid)` 唯一
- 因此 REST 只提供单条查询，不提供列表接口

## 5. 聚合 K 线

### `POST /api/v1/list_klines`

查询聚合 K 线。

请求字段：

- `symbol_guid`，必填
- `timeframe`，当前仅支持 `1m`，也可传空字符串
- `start_timestamp`
- `end_timestamp`
- `pagination.page`
- `pagination.page_size`
- `only_active`

请求示例：

```json
{
  "symbol_guid": "symbol-guid",
  "timeframe": "1m",
  "start_timestamp": 1712900000000,
  "end_timestamp": 1712903600000,
  "pagination": {
    "page": 1,
    "page_size": 100
  },
  "only_active": true
}
```

响应示例：

```json
{
  "code": 200,
  "message": "list klines success",
  "pagination": {
    "page": 1,
    "page_size": 100,
    "total": 2
  },
  "result": [
    {
      "guid": "kline-guid-1",
      "symbol_guid": "symbol-guid",
      "timeframe": "1m",
      "open_price": "65000",
      "close_price": "65010",
      "high_price": "65020",
      "low_price": "64990",
      "volume": "1000",
      "market_cap": "999999",
      "is_active": true,
      "created_at": 1712900000000,
      "updated_at": 1712900060000
    }
  ]
}
```

## 6. 交易所维度 K 线

### `POST /api/v1/list_exchange_klines`

查询交易所维度 K 线。

请求字段：

- `symbol_guid`，必填
- `exchange_guid`，可选
- `timeframe`，当前仅支持 `1m`
- `start_timestamp`
- `end_timestamp`
- `pagination.page`
- `pagination.page_size`
- `only_active`

请求示例：

```json
{
  "symbol_guid": "symbol-guid",
  "exchange_guid": "exchange-guid",
  "timeframe": "1m",
  "start_timestamp": 1712900000000,
  "end_timestamp": 1712903600000,
  "pagination": {
    "page": 1,
    "page_size": 100
  },
  "only_active": true
}
```

响应示例：

```json
{
  "code": 200,
  "message": "list exchange klines success",
  "pagination": {
    "page": 1,
    "page_size": 100,
    "total": 2
  },
  "result": [
    {
      "guid": "exchange-kline-guid-1",
      "exchange_guid": "exchange-guid",
      "symbol_guid": "symbol-guid",
      "timeframe": "1m",
      "open_price": "65000",
      "close_price": "65010",
      "high_price": "65020",
      "low_price": "64990",
      "volume": "1000",
      "market_cap": "999999",
      "is_active": true,
      "created_at": 1712900000000,
      "updated_at": 1712900060000
    }
  ]
}
```

## 7. 参数与实现说明

- 所有金额、价格、成交量、市值字段均使用字符串返回，避免精度损失
- 时间字段统一为 Unix 毫秒时间戳
- `list_klines` 与 `list_exchange_klines` 当前仅支持 `1m`
- `symbol_market` 当前查询“最新记录”时按 `updated_at DESC, created_at DESC, guid DESC` 取第一条
- `get_symbol_market_currency` 默认只查询启用中的记录

