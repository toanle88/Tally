.PHONY: \
	db-config \
	db-up \
	db-wait \
	db-status \
	db-logs \
	db-shell \
	db-down \
	db-version

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

.PHONY: db-migrate db-seed db-verify db-prepare

db-migrate: db-wait
	./scripts/db/migrate.sh

db-seed: db-wait
	./scripts/db/seed.sh

db-verify: db-wait
	./scripts/db/verify.sh

db-prepare: db-migrate db-seed db-verify