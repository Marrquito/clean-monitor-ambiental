CREATE TABLE IF NOT EXISTS measurements (
    id          BIGSERIAL PRIMARY KEY,
    measured_at TIMESTAMPTZ      NOT NULL UNIQUE,
    temperature DOUBLE PRECISION NOT NULL,
    humidity    DOUBLE PRECISION NOT NULL,
    light       DOUBLE PRECISION NOT NULL,
    source      TEXT             NOT NULL DEFAULT 'simulated',
    created_at  TIMESTAMPTZ      NOT NULL DEFAULT now()
);
