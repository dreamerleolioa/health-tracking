-- =====================
-- body_metrics
-- =====================
ALTER TABLE body_metrics ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

DO $$
DECLARE v_uid UUID;
BEGIN
    SELECT id INTO v_uid FROM users LIMIT 1;
    IF v_uid IS NOT NULL THEN
        UPDATE body_metrics SET user_id = v_uid WHERE user_id IS NULL;
    END IF;
END $$;

-- Delete rows that could not be assigned a user (no users exist yet)
DELETE FROM body_metrics WHERE user_id IS NULL;

ALTER TABLE body_metrics ALTER COLUMN user_id SET NOT NULL;
CREATE INDEX idx_body_metrics_user_id ON body_metrics (user_id);

-- =====================
-- sleep_logs
-- =====================
ALTER TABLE sleep_logs ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

DO $$
DECLARE v_uid UUID;
BEGIN
    SELECT id INTO v_uid FROM users LIMIT 1;
    IF v_uid IS NOT NULL THEN
        UPDATE sleep_logs SET user_id = v_uid WHERE user_id IS NULL;
    END IF;
END $$;

DELETE FROM sleep_logs WHERE user_id IS NULL;

ALTER TABLE sleep_logs ALTER COLUMN user_id SET NOT NULL;
CREATE INDEX idx_sleep_logs_user_id ON sleep_logs (user_id);

-- =====================
-- daily_activities（特殊序列：重建 UNIQUE constraint）
-- =====================
ALTER TABLE daily_activities ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;

DO $$
DECLARE v_uid UUID;
BEGIN
    SELECT id INTO v_uid FROM users LIMIT 1;
    IF v_uid IS NOT NULL THEN
        UPDATE daily_activities SET user_id = v_uid WHERE user_id IS NULL;
    END IF;
END $$;

DELETE FROM daily_activities WHERE user_id IS NULL;

ALTER TABLE daily_activities ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE daily_activities DROP CONSTRAINT daily_activities_activity_date_key;
ALTER TABLE daily_activities ADD CONSTRAINT daily_activities_user_id_activity_date_key UNIQUE (user_id, activity_date);
CREATE INDEX idx_daily_activities_user_id ON daily_activities (user_id);
