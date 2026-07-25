-- +goose Up
CREATE TABLE platform.local_seed_manifest (
    seed_name text NOT NULL,
    seed_version bigint NOT NULL,
    checksum text NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT local_seed_manifest_pk
        PRIMARY KEY (seed_name, seed_version),

    CONSTRAINT local_seed_manifest_seed_name_not_blank
        CHECK (btrim(seed_name) <> ''),

    CONSTRAINT local_seed_manifest_seed_version_positive
        CHECK (seed_version > 0),

    CONSTRAINT local_seed_manifest_checksum_not_blank
        CHECK (btrim(checksum) <> '')
);

-- +goose Down
DROP TABLE IF EXISTS platform.local_seed_manifest;