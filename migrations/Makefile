# ============================================================
# Migration Makefile (RINCO)
# ============================================================
# Targets:
#   make migrate            apply SQL + CQL + CH + Mongo
#   make migrate-sql        apply Postgres migrations (Goose)
#   make migrate-cql        apply Scylla migrations
#   make migrate-ch         apply ClickHouse migrations
#   make migrate-mongo      apply Mongo migrations
#   make migrate-status     show pending SQL migrations
#   make migrate-rollback   roll back one SQL migration
#   make migrate-reset      drop + recreate everything
# ============================================================

ROOT := $(shell pwd)
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
POSTGRES_DB   ?= rinco
POSTGRES_USER ?= rinco
POSTGRES_PASSWORD ?= rinco_dev_password

SCYLLA_HOST  ?= localhost:9042
CLICKHOUSE_HOST ?= localhost:8123
MONGO_HOST   ?= localhost:27017

DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: migrate migrate-sql migrate-cql migrate-ch migrate-mongo status rollback reset

migrate: migrate-sql migrate-cql migrate-ch migrate-mongo
	@echo "✓ all migrations applied"

migrate-sql:
	@echo "==> Postgres SQL migrations"
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/migrate -dir sql -cmd up

migrate-cql:
	@echo "==> ScyllaDB CQL migrations"
	for f in $(ROOT)/migrations/cql/*.cql; do \
	  echo "  $$f"; \
	  cqlsh $(SCYLLA_HOST) -f $$f || exit 1; \
	done

migrate-ch:
	@echo "==> ClickHouse migrations"
	for f in $(ROOT)/migrations/ch/*.sql; do \
	  echo "  $$f"; \
	  clickhouse-client --host $(CLICKHOUSE_HOST) --multiquery < $$f || exit 1; \
	done

migrate-mongo:
	@echo "==> MongoDB migrations"
	for f in $(ROOT)/migrations/mongo/*.js; do \
	  echo "  $$f"; \
	  mongosh --quiet --host $(MONGO_HOST) -u $(POSTGRES_USER) -p $(POSTGRES_PASSWORD) \
	    --authenticationDatabase admin rinco < $$f || exit 1; \
	done

status:
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/migrate -dir sql -cmd status

rollback:
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/migrate -dir sql -cmd down

reset:
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/migrate -dir sql -cmd reset
