CREATE UNIQUE INDEX IF NOT EXISTS idx_symbol_market_currency_unique
    ON symbol_market_currency (symbol_guid, currency_guid);
