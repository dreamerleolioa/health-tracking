BEGIN;

-- daily_activities
ALTER TABLE daily_activities DROP CONSTRAINT IF EXISTS daily_activities_user_id_activity_date_key;
ALTER TABLE daily_activities ADD CONSTRAINT daily_activities_activity_date_key UNIQUE (activity_date);
DROP INDEX IF EXISTS idx_daily_activities_user_id;
ALTER TABLE daily_activities DROP COLUMN IF EXISTS user_id;

-- sleep_logs
DROP INDEX IF EXISTS idx_sleep_logs_user_id;
ALTER TABLE sleep_logs DROP COLUMN IF EXISTS user_id;

-- body_metrics
DROP INDEX IF EXISTS idx_body_metrics_user_id;
ALTER TABLE body_metrics DROP COLUMN IF EXISTS user_id;

COMMIT;
