# DLV-PLAT-002 Local Database Reproducibility Verification

- Commit:
- Verification date:
- Operating system:
- Docker version:
- Docker Compose version:
- PostgreSQL image reference:
- Resolved PostgreSQL version:
- Compose project/service:
- Local configuration source: `.env` created from `.env.example`
- Initial volume state: Absent

## Commands executed

1. `docker compose down --volumes`
2. `make db-config`
3. `make db-reset`
4. `make db-reset`

## Results

- Configuration validation result:
- Startup and health result:
- Migration result:
- Seed version:
- Seed checksum:
- Prepared-state verification result:
- First reset result:
- Second reset result:
- Verification results identical: Yes
- Unrelated Docker resources affected: No
- Real credentials required: No
- Azure or external services required: No
- Overall result: PASS | FAIL

## Notes