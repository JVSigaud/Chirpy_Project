-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens(token,created_at,updated_at,expires_at,revoked_at,user_id)
VALUES(
    $1,
    NOW(),
    NOW(),
    NOW()+ interval '60 days',
    NULL,
    $2
);

-- name: LookUpToken :one
SELECT users.* FROM users
JOIN refresh_tokens ON users.id = refresh_tokens.user_id
WHERE refresh_tokens.token = $1
AND revoked_at IS NULL
AND expires_at > NOW();
-- SELECT user_id,expires_at,revoked_at from refresh_tokens
-- where  token  = $1;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens SET revoked_at = NOW(),
updated_at = NOW()
WHERE token = $1
RETURNING *;