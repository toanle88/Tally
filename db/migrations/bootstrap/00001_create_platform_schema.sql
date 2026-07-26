-- +goose Up
CREATE SCHEMA IF NOT EXISTS platform;

-- +goose Down
DROP SCHEMA IF EXISTS platform;
