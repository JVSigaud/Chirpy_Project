-- name: Reset :exec
DELETE FROM users;

-- TRUNCATE TABLE users; mais rapido, sem rollback

