-- =====================
-- Extension
-- =====================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================
-- ENUM Types
-- =====================
CREATE TYPE commute_mode AS ENUM ('scooter', 'train', 'walk', 'other');

-- =====================
-- 體位數據
-- =====================
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

CREATE INDEX idx_body_metrics_recorded_at ON body_metrics (recorded_at DESC);

-- =====================
-- 睡眠紀錄
-- =====================
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

CREATE OR REPLACE FUNCTION set_abnormal_wake()
RETURNS TRIGGER AS $$
BEGIN
    NEW.abnormal_wake := (
        EXTRACT(HOUR FROM NEW.wake_at AT TIME ZONE 'Asia/Taipei') = 3
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_abnormal_wake
BEFORE INSERT OR UPDATE ON sleep_logs
FOR EACH ROW EXECUTE FUNCTION set_abnormal_wake();

CREATE INDEX idx_sleep_logs_sleep_at ON sleep_logs (sleep_at DESC);
CREATE INDEX idx_sleep_logs_abnormal ON sleep_logs (abnormal_wake) WHERE abnormal_wake = TRUE;

-- =====================
-- 每日活動
-- =====================
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

CREATE INDEX idx_daily_activities_date ON daily_activities (activity_date DESC);

-- =====================
-- 魔術練習
-- =====================
CREATE TABLE magic_practices (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    technique_name   TEXT        NOT NULL,
    practiced_at     DATE        NOT NULL,
    proficiency      SMALLINT    CHECK (proficiency BETWEEN 1 AND 5),
    duration_minutes INT         CHECK (duration_minutes > 0),
    video_url        TEXT,
    note             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_magic_practices_technique ON magic_practices (technique_name, practiced_at DESC);

-- =====================
-- MapleStory 角色快照
-- =====================
CREATE TABLE maple_snapshots (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    character_name   TEXT        NOT NULL,
    job              TEXT        NOT NULL,
    level            SMALLINT    NOT NULL CHECK (level BETWEEN 1 AND 300),
    stats            JSONB       NOT NULL DEFAULT '{}',
    snapshot_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note             TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_maple_snapshots_char  ON maple_snapshots (character_name, snapshot_at DESC);
CREATE INDEX idx_maple_snapshots_stats ON maple_snapshots USING gin (stats);

-- =====================
-- updated_at 自動更新觸發器
-- =====================
CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_body_metrics_updated_at
    BEFORE UPDATE ON body_metrics
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_sleep_logs_updated_at
    BEFORE UPDATE ON sleep_logs
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_daily_activities_updated_at
    BEFORE UPDATE ON daily_activities
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TRIGGER trg_magic_practices_updated_at
    BEFORE UPDATE ON magic_practices
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
