CREATE TABLE if not exists model_predictions
(
    id                UUID PRIMARY KEY,
    created_at        TIMESTAMPTZ      NOT NULL,
    zone              TEXT             NOT NULL,
    should_water      BOOLEAN          NOT NULL,
    predicted_seconds DOUBLE PRECISION NOT NULL,
    decision_reason   TEXT             NOT NULL,
    moisture_before   DOUBLE PRECISION NOT NULL,
    watering_executed BOOLEAN          NOT NULL,
    validation_at     TIMESTAMPTZ,
    validation_status TEXT NOT NULL,
    moisture_after    DOUBLE PRECISION,
    target_moisture   DOUBLE PRECISION NOT NULL,
    reached_target    BOOLEAN
);

CREATE UNIQUE INDEX ux_model_predictions_one_pending_per_zone
    ON model_predictions (zone)
    WHERE validation_at IS NULL;

CREATE INDEX idx_model_predictions_zone_created_at
    ON model_predictions (zone, created_at DESC);