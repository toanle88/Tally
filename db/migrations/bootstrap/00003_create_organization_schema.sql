-- +goose Up
CREATE SCHEMA IF NOT EXISTS organization;

-- +goose Down
DROP SCHEMA IF EXISTS organization;
