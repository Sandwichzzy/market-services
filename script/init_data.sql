INSERT INTO asset (asset_name, asset_symbol, asset_logo, is_active) VALUES
    ('Bitcoin', 'BTC', 'https://cryptologos.cc/logos/bitcoin-btc-logo.png', TRUE),
    ('Ethereum', 'ETH', 'https://cryptologos.cc/logos/ethereum-eth-logo.png', TRUE),
    ('Tether USD', 'USDT', 'https://cryptologos.cc/logos/tether-usdt-logo.png', TRUE);


INSERT INTO exchange (name, config, is_active) VALUES
       ('Binance', '{}'::jsonb, TRUE),
       ('OKX', '{}'::jsonb, TRUE),
       ('Bybit', '{}'::jsonb, TRUE);


INSERT INTO symbol (
    symbol_name,
    base_asset_guid,
    qoute_asset_guid,
    market_type,
    is_active
)
VALUES
    (
        'BTC/USDT',
        (SELECT guid FROM asset WHERE asset_symbol = 'BTC' LIMIT 1),
        (SELECT guid FROM asset WHERE asset_symbol = 'USDT' LIMIT 1),
        'SPOT',
        TRUE
    ),
    (
        'ETH/USDT',
        (SELECT guid FROM asset WHERE asset_symbol = 'ETH' LIMIT 1),
        (SELECT guid FROM asset WHERE asset_symbol = 'USDT' LIMIT 1),
        'SPOT',
        TRUE
    );


INSERT INTO exchange_symbol (
    exchange_guid,
    symbol_guid,
    price,
    ask_price,
    bid_price,
    volume,
    radio,
    is_active
)
VALUES
-- Binance
(
    '8a6d4b7bfaeb482090a92940fd0ff0c4',
    '044fd58b71bb4dcca80c8b0b6c8a8ca6',
    68500.120000000000000000,
    68510.500000000000000000,
    68495.800000000000000000,
    1250,
    0.023500000000000000,
    TRUE
),
(
    '8a6d4b7bfaeb482090a92940fd0ff0c4',
    'e06a350484ac48c492f083d40f7d58e4',
    3520.450000000000000000,
    3521.100000000000000000,
    3519.900000000000000000,
    2860,
    0.018200000000000000,
    TRUE
),
-- OKX
(
    '2f95152b4ab24b45918311edf9330c28',
    '044fd58b71bb4dcca80c8b0b6c8a8ca6',
    68498.900000000000000000,
    68508.200000000000000000,
    68493.600000000000000000,
    980,
    0.021800000000000000,
    TRUE
),
(
    '2f95152b4ab24b45918311edf9330c28',
    'e06a350484ac48c492f083d40f7d58e4',
    3518.700000000000000000,
    3519.500000000000000000,
    3518.200000000000000000,
    2140,
    0.016500000000000000,
    TRUE
),

-- Bybit
(
    '157aa7ecfefd4606a5e5b921192ac3d9',
    '044fd58b71bb4dcca80c8b0b6c8a8ca6',
    68502.300000000000000000,
    68512.000000000000000000,
    68497.100000000000000000,
    1105,
    0.022600000000000000,
    TRUE
),
(
    '157aa7ecfefd4606a5e5b921192ac3d9',
    'e06a350484ac48c492f083d40f7d58e4',
    3521.200000000000000000,
    3521.800000000000000000,
    3520.600000000000000000,
    1988,
    0.017300000000000000,
    TRUE
);