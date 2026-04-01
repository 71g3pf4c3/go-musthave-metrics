CREATE TABLE IF NOT EXISTS metrics (
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    value_double DOUBLE PRECISION,
    value_bigint BIGINT,
    CONSTRAINT metrics_kind_check CHECK (kind IN ('gauge', 'counter')),
    CONSTRAINT metrics_value_check CHECK (
        (kind = 'gauge' AND value_double IS NOT NULL AND value_bigint IS NULL) OR
        (kind = 'counter' AND value_bigint IS NOT NULL AND value_double IS NULL)
    ),
    PRIMARY KEY (name, kind)
);
