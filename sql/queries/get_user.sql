-- name: GetUser :one
SELECT * FROM users
WHERE email = $1;

-- name: UpgradeMember :exec
UPDATE users 
SET is_chirpy_red = TRUE
WHERE id = $1;