\set ON_ERROR_STOP on

BEGIN;

INSERT INTO platform.local_seed_manifest (
    seed_name,
    seed_version,
    checksum
)
VALUES (
    :'seed_name',
    :'seed_version'::bigint,
    :'checksum'
)
ON CONFLICT (seed_name, seed_version) DO NOTHING;

SELECT
    seed_version = :'seed_version'::bigint
    AND checksum = :'checksum' AS seed_matches
FROM platform.local_seed_manifest
WHERE seed_name = :'seed_name'
\gset

\if :seed_matches
\else
    \echo 'Seed conflict: content changed without a version change.'
    \quit 1
\endif

COMMIT;