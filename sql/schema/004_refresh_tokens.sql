-- +goose Up
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE refresh_tokens (
--   id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  token TEXT PRIMARY KEY,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  revoked_at TIMESTAMP DEFAULT NULL,
  user_id UUID NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE refresh_tokens;