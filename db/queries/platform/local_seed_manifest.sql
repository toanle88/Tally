-- name: GetLocalSeedManifestChecksum :one
SELECT checksum
FROM platform.local_seed_manifest
WHERE seed_name = sqlc.arg(seed_name)
  AND seed_version = sqlc.arg(seed_version);
