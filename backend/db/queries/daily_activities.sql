-- name: CreateDailyActivity :one
INSERT INTO daily_activities (
    activity_date, steps, commute_mode, commute_minutes, note
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetDailyActivity :one
SELECT * FROM daily_activities WHERE id = $1;

-- name: GetDailyActivityByDate :one
SELECT * FROM daily_activities WHERE activity_date = $1;

-- name: ListDailyActivities :many
SELECT * FROM daily_activities
WHERE
    (sqlc.narg('from')::DATE IS NULL OR activity_date >= sqlc.narg('from')::DATE)
    AND (sqlc.narg('to')::DATE IS NULL OR activity_date <= sqlc.narg('to')::DATE)
ORDER BY activity_date DESC
LIMIT sqlc.arg('limit');

-- name: UpdateDailyActivity :one
UPDATE daily_activities
SET
    steps           = COALESCE(sqlc.narg('steps'), steps),
    commute_mode    = COALESCE(sqlc.narg('commute_mode'), commute_mode),
    commute_minutes = COALESCE(sqlc.narg('commute_minutes'), commute_minutes),
    note            = COALESCE(sqlc.narg('note'), note),
    updated_at      = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteDailyActivity :exec
DELETE FROM daily_activities WHERE id = $1;
