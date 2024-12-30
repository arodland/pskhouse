CREATE TABLE pskhouse.rx
(
    `id` UInt64,
    `time` DateTime,
    `band` Int16,
    `mode` LowCardinality(String),
    `rx_sign` String,
    `rx_lat` Float32,
    `rx_lon` Float32,
    `rx_loc` String,
    `tx_sign` String,
    `tx_lat` Float32,
    `tx_lon` Float32,
    `tx_loc` String,
    `distance` UInt16,
    `azimuth` UInt16,
    `rx_azimuth` UInt16,
    `frequency` UInt32,
    `snr` Int8,
    `version` String
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(time)
ORDER BY time;
