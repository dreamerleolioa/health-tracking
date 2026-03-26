-- name: CreateSleepLog :one
INSERT INTO sleep_logs (
    user_id, sleep_at, wake_at, quality, note
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetSleepLog :one
SELECT * FROM sleep_logs WHERE id = $1 AND user_id = $2;

-- name: ListSleepLogs :many
SELECT * FROM sleep_logs
WHERE
    user_id = sqlc.arg('user_id')
    AND (sqlc.narg('from')::DATE IS NULL OR (wake_at AT TIME ZONE 'Asia/Taipei')::DATE >= sqlc.narg('from')::DATE)
    AND (sqlc.narg('to')::DATE IS NULL OR (wake_at AT TIME ZONE 'Asia/Taipei')::DATE <= sqlc.narg('to')::DATE)
    AND (sqlc.narg('abnormal_only')::BOOLEAN IS NULL OR abnormal_wake = sqlc.narg('abnormal_only'))
ORDER BY wake_at DESC
LIMIT sqlc.arg('limit');

-- name: UpdateSleepLog :one
UPDATE sleep_logs
SET
    sleep_at   = COALESCE(sqlc.narg('sleep_at'), sleep_at),
    wake_at    = COALESCE(sqlc.narg('wake_at'), wake_at),
    quality    = COALESCE(sqlc.narg('quality'), quality),
    note       = COALESCE(sqlc.narg('note'), note),
    updated_at = NOW()
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteSleepLog :exec
DELETE FROM sleep_logs WHERE id = $1 AND user_id = $2;
