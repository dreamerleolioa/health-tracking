-- name: CreateSleepLog :one
INSERT INTO sleep_logs (
    sleep_at, wake_at, quality, note
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetSleepLog :one
SELECT * FROM sleep_logs WHERE id = $1;

-- name: ListSleepLogs :many
SELECT * FROM sleep_logs
WHERE
    -- Filter by wake_at Taipei date (user thinks of sleep by the morning they woke up)
    ($1::DATE IS NULL OR (wake_at AT TIME ZONE 'Asia/Taipei')::DATE >= $1::DATE)
    AND ($2::DATE IS NULL OR (wake_at AT TIME ZONE 'Asia/Taipei')::DATE <= $2::DATE)
    AND ($3::BOOLEAN IS NULL OR ($3 = FALSE) OR abnormal_wake = TRUE)
ORDER BY wake_at DESC
LIMIT $4;

-- name: UpdateSleepLog :one
UPDATE sleep_logs
SET
    sleep_at   = COALESCE(sqlc.narg('sleep_at'), sleep_at),
    wake_at    = COALESCE(sqlc.narg('wake_at'), wake_at),
    quality    = COALESCE(sqlc.narg('quality'), quality),
    note       = COALESCE(sqlc.narg('note'), note),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteSleepLog :exec
DELETE FROM sleep_logs WHERE id = $1;
