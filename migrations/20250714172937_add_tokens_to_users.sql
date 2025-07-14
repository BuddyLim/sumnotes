-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ADD COLUMN access_token TEXT,
ADD COLUMN refresh_token TEXT,
ADD COLUMN token_expiry TIMESTAMP
WITH
    TIME ZONE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
DROP COLUMN access_token,
DROP COLUMN refresh_token,
DROP COLUMN token_expiry;
-- +goose StatementEnd