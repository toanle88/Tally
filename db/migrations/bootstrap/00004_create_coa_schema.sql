-- +goose Up
CREATE SCHEMA IF NOT EXISTS coa;

-- +goose Down
DROP SCHEMA IF EXISTS coa;
