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

CREATE TRIGGER trg_magic_practices_updated_at
    BEFORE UPDATE ON magic_practices
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

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
