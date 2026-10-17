-- +goose Up
CREATE SCHEMA IF NOT EXISTS invoicing;

-- +goose Down
DROP SCHEMA IF EXISTS invoicing CASCADE;
