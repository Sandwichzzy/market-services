DO
$$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'exchange_symbol_kline' AND column_name = 'low_active'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'exchange_symbol_kline' AND column_name = 'low_price'
    ) THEN
        ALTER TABLE exchange_symbol_kline RENAME COLUMN low_active TO low_price;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'exchange_symbol_kline' AND column_name = 'low_price'
    ) THEN
        ALTER TABLE exchange_symbol_kline
            ADD COLUMN low_price NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (low_price >= 0);
    END IF;
END
$$;

DO
$$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'symbol_kline' AND column_name = 'low_active'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'symbol_kline' AND column_name = 'low_price'
    ) THEN
        ALTER TABLE symbol_kline RENAME COLUMN low_active TO low_price;
    ELSIF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'symbol_kline' AND column_name = 'low_price'
    ) THEN
        ALTER TABLE symbol_kline
            ADD COLUMN low_price NUMERIC(65, 18) NOT NULL DEFAULT 0 CHECK (low_price >= 0);
    END IF;
END
$$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_exchange_symbol_kline_unique
    ON exchange_symbol_kline (exchange_guid, symbol_guid, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_symbol_kline_unique
    ON symbol_kline (symbol_guid, created_at);
