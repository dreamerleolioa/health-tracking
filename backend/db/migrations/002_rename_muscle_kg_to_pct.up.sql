-- Rename muscle_kg to muscle_pct if the old column still exists.
-- Migration 001 was updated in-place to use muscle_pct directly,
-- so this is a no-op on fresh installs.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'body_metrics' AND column_name = 'muscle_kg'
    ) THEN
        ALTER TABLE body_metrics RENAME COLUMN muscle_kg TO muscle_pct;
    END IF;
END $$;
