-- name: UpsertUser :one
INSERT INTO users (google_id, email, display_name, avatar_url, last_login_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (google_id) DO UPDATE SET
    email         = EXCLUDED.email,
    display_name  = EXCLUDED.display_name,
    avatar_url    = EXCLUDED.avatar_url,
    last_login_at = NOW()
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = $1;
