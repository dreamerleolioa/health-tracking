-- Final schema state for sqlc code generation.
-- This represents the current DB state after all migrations.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE commute_mode AS ENUM ('scooter', 'train', 'walk', 'other');

CREATE TABLE body_metrics (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    weight_kg     NUMERIC(5,2),
    body_fat_pct  NUMERIC(5,2),
    muscle_pct    NUMERIC(5,2),
    visceral_fat  SMALLINT    CHECK (visceral_fat BETWEEN 1 AND 30),
    recorded_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note          TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sleep_logs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    sleep_at        TIMESTAMPTZ NOT NULL,
    wake_at         TIMESTAMPTZ NOT NULL,
    duration_min    INT         GENERATED ALWAYS AS (
                        (EXTRACT(EPOCH FROM (wake_at - sleep_at)) / 60)::INT
                    ) STORED,
    abnormal_wake   BOOLEAN     NOT NULL DEFAULT FALSE,
    quality         SMALLINT    CHECK (quality BETWEEN 1 AND 5),
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE daily_activities (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_date    DATE         NOT NULL UNIQUE,
    steps            INT          CHECK (steps >= 0),
    commute_mode     commute_mode,
    commute_minutes  INT          CHECK (commute_minutes >= 0),
    note             TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
