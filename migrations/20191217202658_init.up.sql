CREATE TABLE telemetry_batch
(
    id          VARCHAR PRIMARY KEY,
    vin         VARCHAR(17) NOT NULL,
    event_count INT NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);
