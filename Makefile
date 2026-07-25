.PHONY: db-up db-wait db-status db-version db-down

db-up:
	docker compose up --detach postgres
	$(MAKE) db-wait

db-wait:
	@set -eu; \
	container_id="$$(docker compose ps --quiet postgres)"; \
	if [ -z "$$container_id" ]; then \
		echo "PostgreSQL container is not running." >&2; \
		docker compose ps postgres >&2 || true; \
		exit 1; \
	fi; \
	echo "Waiting for PostgreSQL to become healthy..."; \
	attempt=1; \
	max_attempts=30; \
	while [ "$$attempt" -le "$$max_attempts" ]; do \
		running="$$(docker inspect --format '{{.State.Running}}' "$$container_id" 2>/dev/null || true)"; \
		status="$$(docker inspect \
			--format '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' \
			"$$container_id" 2>/dev/null || true)"; \
		if [ "$$running" != "true" ]; then \
			echo "PostgreSQL stopped before becoming healthy." >&2; \
			docker compose ps postgres >&2 || true; \
			docker compose logs --tail=100 postgres >&2 || true; \
			exit 1; \
		fi; \
		case "$$status" in \
			healthy) \
				echo "PostgreSQL is healthy."; \
				exit 0; \
				;; \
			unhealthy) \
				echo "PostgreSQL became unhealthy." >&2; \
				docker compose ps postgres >&2 || true; \
				docker compose logs --tail=100 postgres >&2 || true; \
				exit 1; \
				;; \
			missing) \
				echo "PostgreSQL has no configured health check." >&2; \
				exit 1; \
				;; \
		esac; \
		sleep 1; \
		attempt=$$((attempt + 1)); \
	done; \
	echo "Timed out waiting for PostgreSQL health." >&2; \
	docker compose ps postgres >&2 || true; \
	docker compose logs --tail=100 postgres >&2 || true; \
	exit 1

db-status:
	docker compose ps postgres

db-version:
	docker compose exec postgres \
		sh -c 'psql --username "$$POSTGRES_USER" --dbname "$$POSTGRES_DB" --tuples-only --no-align --command "SHOW server_version;"'

db-down:
	docker compose down