ALTER TABLE model_predictions ADD COLUMN validate_after timestamptz;

UPDATE model_predictions
SET validate_after = created_at + interval '15 minutes'
WHERE validate_after IS NULL;

ALTER TABLE model_predictions
    ALTER COLUMN validate_after SET NOT NULL;