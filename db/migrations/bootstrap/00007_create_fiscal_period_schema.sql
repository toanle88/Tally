-- +goose Up
CREATE SCHEMA IF NOT EXISTS fiscal_period;

-- +goose Down
DROP SCHEMA IF EXISTS fiscal_period CASCADE;
