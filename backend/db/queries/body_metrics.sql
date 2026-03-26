-- name: CreateBodyMetric :one
INSERT INTO body_metrics (
    user_id, weight_kg, body_fat_pct, muscle_pct, visceral_fat, recorded_at, note
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetBodyMetric :one
SELECT * FROM body_metrics WHERE id = $1 AND user_id = $2;

-- name: ListBodyMetrics :many
SELECT * FROM body_metrics
WHERE
    user_id = sqlc.arg('user_id')
    AND (sqlc.narg('from')::DATE IS NULL OR recorded_at::DATE >= sqlc.narg('from')::DATE)
    AND (sqlc.narg('to')::DATE IS NULL OR recorded_at::DATE <= sqlc.narg('to')::DATE)
ORDER BY recorded_at DESC
LIMIT sqlc.arg('limit');

-- name: UpdateBodyMetric :one
UPDATE body_metrics
SET
    weight_kg    = COALESCE(sqlc.narg('weight_kg'), weight_kg),
    body_fat_pct = COALESCE(sqlc.narg('body_fat_pct'), body_fat_pct),
    muscle_pct   = COALESCE(sqlc.narg('muscle_pct'), muscle_pct),
    visceral_fat = COALESCE(sqlc.narg('visceral_fat'), visceral_fat),
    note         = COALESCE(sqlc.narg('note'), note),
    updated_at   = NOW()
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteBodyMetric :exec
DELETE FROM body_metrics WHERE id = $1 AND user_id = $2;
