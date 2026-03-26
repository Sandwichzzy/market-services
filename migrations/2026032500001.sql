-- ============================================
-- 行情服务数据库初始化脚本
-- Market Services System Database Schema
-- ============================================

-- 创建自定义类型：UINT256（无符号 256 位整数）
DO
$$
BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'uint256') THEN
            CREATE DOMAIN UINT256 AS NUMERIC CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
        ELSE
            ALTER DOMAIN UINT256 DROP CONSTRAINT uint256_check;
            ALTER DOMAIN UINT256 ADD CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
        END IF;
END
$$;

-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" CASCADE;


-- ============================================
-- 资产配置表 (Asset)
-- ============================================
CREATE TABLE IF NOT EXISTS asset (
    guid                 TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),  -- 主键：资产唯一标识
    asset_name           VARCHAR(100) NOT NULL DEFAULT 'Dollar',  -- 资产名称（如：Dollar、Bitcoin）
    asset_symbol         VARCHAR(20)  NOT NULL DEFAULT 'USD',     -- 资产符号（如：USD、BTC、ETH）
    asset_logo           VARCHAR(500) NOT NULL,                   -- 资产图标 URL
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,           -- 是否启用（新增字段）
    created_at           TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP,  -- 创建时间
    updated_at           TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP  -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_asset_guid ON asset (guid);
CREATE INDEX IF NOT EXISTS idx_asset_symbol ON asset (asset_symbol);  -- 新增：按符号查询
CREATE INDEX IF NOT EXISTS idx_asset_is_active ON asset (is_active);  -- 新增：按状态查询


-- --------------------------------------------
-- 交易所表 (exchange)
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS exchange(
    guid               TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''),  -- 交易所唯一标识
    name               VARCHAR(100) NOT NULL UNIQUE,             -- 交易所名称（唯一），如：Binance、OKX、ByteLink事件预测平台
    config             JSONB,                                    -- 交易所配置（JSON格式），存储API Key、Secret、Endpoint等敏感信息
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,            -- 是否启用（新增字段）
    created_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP,   -- 创建时间
    updated_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP    -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_exchange_guid ON exchange (guid);
CREATE INDEX IF NOT EXISTS idx_exchange_is_active ON exchange (is_active);
CREATE INDEX IF NOT EXISTS idx_exchange_created_at ON exchange (created_at);

-- --------------------------------------------
-- 交易对表 (symbol)
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS symbol(
    guid               TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''), -- 主键：交易对唯一标识
    symbol_name        VARCHAR(100) NOT NULL,                     -- 交易对名称（如 BTC/USDT）
    base_asset_guid    VARCHAR(100) NOT NULL,                     -- 基础资产ID（如 BTC）
    qoute_asset_guid   VARCHAR(100) NOT NULL,                     -- 计价资产ID（如 USDT）
    market_type        VARCHAR(100) NOT NULL DEFAULT 'SPOT',      -- 市场类型（SPOT:现货,FUTURE:期货,OPTION:期权）
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,             -- 是否启用
    created_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP,    -- 创建时间
    updated_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP     -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_symbol_guid ON symbol (guid);
CREATE INDEX IF NOT EXISTS idx_symbol_is_active ON symbol (is_active);
CREATE INDEX IF NOT EXISTS idx_symbol_created_at ON symbol (created_at);

-- --------------------------------------------
-- 法币汇率表 (currency)
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS currency(
    guid               TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''), -- 主键：法币唯一标识
    currency_name      VARCHAR(100) NOT NULL,                -- 法币名称（如 US Dollar）
    currency_code      VARCHAR(100) NOT NULL,                -- 法币代码（如 USD、CNY）
    rate               NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (rate >= 0),         -- 汇率（相对基准货币）
    buy_spread         NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (buy_spread >= 0),   -- 买入价差（加点）
    sell_spread        NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (sell_spread >= 0),   -- 卖出价差（加点）
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,         -- 是否启用
    created_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP,-- 创建时间
    updated_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_currency_guid ON currency (guid);
CREATE INDEX IF NOT EXISTS idx_currency_is_active ON currency (is_active);
CREATE INDEX IF NOT EXISTS idx_currency_created_at ON currency (created_at);


-- --------------------------------------------
-- 交易所和交易对关联表 (exchange_symbol)
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS exchange_symbol(
    guid               TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''), -- 主键：唯一标识
    exchange_guid      VARCHAR(100) NOT NULL,             -- 交易所ID（关联 exchange 表）
    symbol_guid        VARCHAR(100) NOT NULL,             -- 交易对ID（关联 symbol 表）
    price              NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (price >= 0),     -- 最新成交价
    ask_price          NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (ask_price >= 0), -- 卖一价（最优卖价）
    bid_price          NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (bid_price >= 0), -- 买一价（最优买价）
    volume             UINT256 NOT NULL,                  -- 成交量（大整数，防止溢出）
    radio              NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (radio >= 0),     -- 涨跌幅（比例值）
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,     -- 是否启用
    created_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP  -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_exchange_symbol_guid ON exchange_symbol (guid);
CREATE INDEX IF NOT EXISTS idx_exchange_symbol_is_active ON exchange_symbol (is_active);
CREATE INDEX IF NOT EXISTS idx_exchange_symbol_created_at ON exchange_symbol (created_at);

-- --------------------------------------------
-- exchange_symbol_kline
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS exchange_symbol_kline(
    guid               TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''), -- 主键：K线记录唯一标识
    exchange_guid      VARCHAR(100) NOT NULL,             -- 交易所ID
    symbol_guid        VARCHAR(100) NOT NULL,             -- 交易对ID
    open_price         NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (open_price >= 0),  -- 开盘价
    close_price        NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (close_price >= 0),-- 收盘价
    high_price         NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (high_price >= 0), -- 最高价
    low_active         NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (low_active >= 0), -- 最低价
    volume             UINT256 NOT NULL,                  -- 成交量
    market_cap         UINT256 NOT NULL,                  -- 市值或成交额
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP, -- K线时间
    updated_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP  -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_exchange_symbol_kline_guid ON exchange_symbol_kline (guid);
CREATE INDEX IF NOT EXISTS idx_exchange_symbol_kline_is_active ON exchange_symbol_kline (is_active);
CREATE INDEX IF NOT EXISTS idx_exchange_symbol_kline_created_at ON exchange_symbol_kline (created_at);

-- --------------------------------------------
-- symbol_market
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS symbol_market(
    guid               TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''), -- 主键：唯一标识
    symbol_guid        VARCHAR(100) NOT NULL,             -- 交易对ID
    price              NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (price >= 0),     -- 当前价格（全市场聚合）
    ask_price          NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (ask_price >= 0), -- 最优卖价
    bid_price          NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (bid_price >= 0), -- 最优买价
    volume             UINT256 NOT NULL,                  -- 成交量（全市场）
    market_cap         UINT256 NOT NULL,                  -- 市值
    radio              NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (radio >= 0),     -- 涨跌幅
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,     -- 是否启用
    created_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP  -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_symbol_market_guid ON symbol_market (guid);
CREATE INDEX IF NOT EXISTS idx_symbol_market_is_active ON symbol_market (is_active);
CREATE INDEX IF NOT EXISTS idx_symbol_market_created_at ON symbol_market (created_at);

-- --------------------------------------------
-- symbol_market_currency
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS symbol_market_currency(
    guid               TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''), -- 主键：唯一标识
    symbol_guid        VARCHAR(100) NOT NULL,             -- 交易对ID
    currency_guid      VARCHAR(100) NOT NULL,             -- 法币ID（关联 currency 表）
    price              NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (price >= 0),     -- 当前价格（法币计价）
    ask_price          NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (ask_price >= 0), -- 卖一价
    bid_price          NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (bid_price >= 0), -- 买一价
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,     -- 是否启用
    created_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP  -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_symbol_market_currency_guid ON symbol_market_currency (guid);
CREATE INDEX IF NOT EXISTS idx_symbol_market_currency_is_active ON symbol_market_currency (is_active);
CREATE INDEX IF NOT EXISTS idx_symbol_market_currency_created_at ON symbol_market_currency (created_at);

-- --------------------------------------------
-- symbol_kline
-- --------------------------------------------
CREATE TABLE IF NOT EXISTS symbol_kline(
    guid               TEXT PRIMARY KEY DEFAULT replace(uuid_generate_v4()::text, '-', ''), -- 主键：K线唯一标识
    symbol_guid        VARCHAR(100) NOT NULL,             -- 交易对ID
    open_price         NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (open_price >= 0),  -- 开盘价
    close_price        NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (close_price >= 0),-- 收盘价
    high_price         NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (high_price >= 0), -- 最高价
    low_active         NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (low_active >= 0), -- 最低价
    volume             UINT256 NOT NULL,                  -- 成交量
    market_cap         UINT256 NOT NULL,                  -- 市值或成交额
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP, -- K线时间
    updated_at         TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP  -- 更新时间
    );
CREATE INDEX IF NOT EXISTS idx_symbol_kline_guid ON symbol_kline (guid);
CREATE INDEX IF NOT EXISTS idx_symbol_kline_is_active ON symbol_kline (is_active);
CREATE INDEX IF NOT EXISTS idx_symbol_kline_created_at ON symbol_kline (created_at);