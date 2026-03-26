-- Final schema state for sqlc code generation.
-- This represents the current DB state after all migrations.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================
-- 使用者（Google OAuth）
-- =====================
CREATE TABLE users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    google_id       TEXT        NOT NULL UNIQUE,
    email           TEXT        NOT NULL UNIQUE,
    display_name    TEXT,
    avatar_url      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_google_id ON users (google_id);

-- =====================
-- Refresh Token
-- =====================
CREATE TABLE refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

CREATE TYPE commute_mode AS ENUM ('scooter', 'train', 'walk', 'other');

CREATE TABLE body_metrics (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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

CREATE INDEX idx_sleep_logs_sleep_at  ON sleep_logs (sleep_at DESC);
CREATE INDEX idx_sleep_logs_abnormal  ON sleep_logs (abnormal_wake) WHERE abnormal_wake = TRUE;

CREATE TABLE daily_activities (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_date    DATE         NOT NULL,
    UNIQUE (user_id, activity_date),
    steps            INT          CHECK (steps >= 0),
    commute_mode     commute_mode,
    commute_minutes  INT          CHECK (commute_minutes >= 0),
    note             TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_daily_activities_date ON daily_activities (activity_date DESC);
