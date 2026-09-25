-- +goose Up
CREATE SCHEMA IF NOT EXISTS identity;

-- +goose Down
DROP SCHEMA IF EXISTS identity;
