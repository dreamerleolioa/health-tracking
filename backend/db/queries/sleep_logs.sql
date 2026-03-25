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
    (sqlc.narg('from')::DATE IS NULL OR (wake_at AT TIME ZONE 'Asia/Taipei')::DATE >= sqlc.narg('from')::DATE)
    AND (sqlc.narg('to')::DATE IS NULL OR (wake_at AT TIME ZONE 'Asia/Taipei')::DATE <= sqlc.narg('to')::DATE)
    AND (sqlc.narg('abnormal_only')::BOOLEAN IS NULL OR sqlc.narg('abnormal_only') = FALSE OR abnormal_wake = TRUE)
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
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteSleepLog :exec
DELETE FROM sleep_logs WHERE id = $1;
