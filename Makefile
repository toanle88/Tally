.PHONY: \
	db-config \
	db-up \
	db-wait \
	db-status \
	db-logs \
	db-shell \
	db-down \
	db-version \
	db-migrate \
	db-seed \
	db-verify \
	db-prepare \
	db-reset \
	db-migrate-status \
	db-migrate-validate \
	db-migrate-create \
	db-migrate-check \
	db-migrate-inventory \
	check \
	verify-database \
	verify-database-clean \
	db-sqlc-version \
	db-sqlc-generate \
	db-sqlc-check \
	db-sqlc-integration \
	persistence-tools-version \
	sqlc-compile \
	sqlc-check \
	persistence-integration-test \
	persistence-check	


DB_SERVICE := postgres
DB_WAIT_ATTEMPTS := 50
DB_WAIT_INTERVAL_SECONDS := 1

db-config:
	docker compose version >/dev/null 2>&1 || { echo "Docker Compose is required. Install from https://docs.docker.com/compose/install/" >&2; exit 1; }
	docker compose config --quiet

db-up: db-config
	docker compose up --detach $(DB_SERVICE)
	$(MAKE) db-wait

db-wait:
	@attempt=1; \
	while [ "$$attempt" -le "$(DB_WAIT_ATTEMPTS)" ]; do \
		container_id="$$(docker compose ps --quiet $(DB_SERVICE) 2>/dev/null)"; \
		if [ -n "$$container_id" ]; then \
			running="$$(docker inspect --format '{{.State.Running}}' "$$container_id" 2>/dev/null)"; \
			if [ "$$running" != "true" ]; then \
				echo "$(DB_SERVICE) stopped before becoming healthy" >&2; \
				docker compose ps $(DB_SERVICE) >&2; \
				docker compose logs --tail=50 $(DB_SERVICE) >&2; \
				exit 1; \
			fi; \
			status="$$(docker inspect --format '{{.State.Health.Status}}' "$$container_id" 2>/dev/null)" || { echo "$(DB_SERVICE) container removed during health check" >&2; docker compose ps $(DB_SERVICE) >&2; exit 1; }; \
			if [ -z "$$status" ]; then \
				echo "$(DB_SERVICE) has no health check configured" >&2; \
				exit 1; \
			fi; \
			if [ "$$status" = "healthy" ]; then \
				echo "$(DB_SERVICE) is healthy"; \
				exit 0; \
			fi; \
			if [ "$$status" = "unhealthy" ]; then \
				echo "$(DB_SERVICE) became unhealthy" >&2; \
				docker compose ps $(DB_SERVICE) >&2; \
				docker compose logs --tail=50 $(DB_SERVICE) >&2; \
				exit 1; \
			fi; \
		fi; \
		echo "Waiting for $(DB_SERVICE) health ($$attempt/$(DB_WAIT_ATTEMPTS))..."; \
		sleep "$(DB_WAIT_INTERVAL_SECONDS)"; \
		attempt=$$((attempt + 1)); \
	done; \
	echo "Timed out waiting for $(DB_SERVICE) to become healthy" >&2; \
	docker compose ps $(DB_SERVICE) >&2; \
	docker compose logs --tail=50 $(DB_SERVICE) >&2; \
	exit 1

db-status:
	docker compose ps $(DB_SERVICE)

db-logs:
	docker compose logs --tail=100 $(DB_SERVICE)

db-shell:
	docker compose exec --interactive --tty $(DB_SERVICE) \
		sh -c 'exec psql --username "$$POSTGRES_USER" --dbname "$$POSTGRES_DB"'

db-down:
	docker compose down

db-version:
	docker compose exec $(DB_SERVICE) \
		sh -c 'psql --username "$$POSTGRES_USER" --dbname "$$POSTGRES_DB" --tuples-only --no-align --command "SHOW server_version;"'

db-migrate: db-wait
	./scripts/db/migrate.sh up

db-seed: db-wait
	./scripts/db/seed.sh

db-verify: db-wait
	./scripts/db/verify.sh

db-prepare: db-migrate db-seed db-verify

db-reset:
	@printf '%s\n' \
		'WARNING: db-reset permanently deletes the TALLY local PostgreSQL database volume.'
	docker compose down --volumes
	$(MAKE) db-up
	$(MAKE) db-prepare

db-migrate-status: db-wait
	./scripts/db/migrate.sh status

db-migrate-validate:
	./scripts/db/migrate.sh validate

db-migrate-create:
	@if [ -z "$(SCHEMA)" ] || [ -z "$(NAME)" ]; then \
		echo "Usage: make db-migrate-create SCHEMA=<bootstrap|platform> NAME=<migration_name>" >&2; \
		exit 2; \
	fi
	./scripts/db/migrate.sh create "$(SCHEMA)" "$(NAME)"

db-migrate-check:
	./scripts/db/migrate.sh check

db-migrate-inventory:
	find db/migrations \
		-type f \
		-name '*.sql' \
		-print0 | \
		sort -z | \
		xargs -0 sha256sum > db/migrations/checksums.sha256
	@echo "Updated db/migrations/checksums.sha256"

check: db-migrate-validate db-migrate-check
	go test ./...

verify-database:
	./scripts/verify/database.sh

verify-database-clean:
	./scripts/verify/database.sh --clean

db-sqlc-version:
	@bash ./scripts/tools/sqlc.sh version

db-sqlc-generate:
	@bash ./scripts/tools/sqlc.sh generate -f sqlc.yaml

db-sqlc-check:
	@bash ./scripts/db/sqlc-check.sh

db-sqlc-integration:
	@set -a; \
	. ./.env; \
	set +a; \
	go test -tags=integration ./internal/platform/database \
		-run '^TestSQLCPlatformSeedManifestWithinTransaction$$' \
		-count=1 \
		-v

# Persistence tooling is pinned through the repository-controlled Go tool
# declarations established by earlier DLV-PLAT-003 stories.
GOOSE ?= go tool goose
SQLC ?= go tool sqlc

## Print the exact repository-pinned persistence tool versions.
persistence-tools-version:
	@$(GOOSE) -version
	@$(SQLC) version

## Parse and type-check the committed sqlc schema and query source.
sqlc-compile:
	@$(SQLC) -f sqlc.yaml compile

## Compare committed generated output with deterministic sqlc output.
## This fails for stale, deleted, or manually edited generated files.
sqlc-check:
	@$(SQLC) -f sqlc.yaml diff

## Run all database integration tests, including clean migration and
## generated-query verification, against PostgreSQL 18 Testcontainers.
persistence-integration-test:
	@go test \
		-tags=integration \
		-count=1 \
		-timeout=10m \
		./internal/platform/database

## Run the complete local persistence release gate used by CI.
persistence-check:
	@bash scripts/db/persistence-check.sh