-- +goose Up
CREATE SCHEMA IF NOT EXISTS workflow;

-- +goose Down
DROP SCHEMA IF EXISTS workflow CASCADE;
