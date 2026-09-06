#!/usr/bin/env bash
# ============================================================
# PostgreSQL replication slot setup (primary + replicas)
# ============================================================
# Run once on the PRIMARY after PostgreSQL is initialized.
# Creates publication + replication slots for each replica.
# ============================================================

set -euo pipefail

PRIMARY_HOST=${PRIMARY_HOST:-postgres-primary}
REPLICA_HOSTS=${REPLICA_HOSTS:-"postgres-replica-1 postgres-replica-2"}
PGUSER=${PGUSER:-rinco}
PGDB=${PGDB:-rinco}
PGPASSWORD=${PGPASSWORD:-rinco_dev_password}

export PGPASSWORD

echo "==> Creating publication for logical replication"
psql -h "$PRIMARY_HOST" -U "$PGUSER" -d "$PGDB" -v ON_ERROR_STOP=1 <<'EOF'
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'rinco_pub') THEN
    CREATE PUBLICATION rinco_pub FOR ALL TABLES;
  END IF;
END$$;
EOF

for REPLICA in $REPLICA_HOSTS; do
  echo "==> Creating replication slot for $REPLICA"
  SLOT_NAME="rinco_slot_$(echo "$REPLICA" | tr -dc 'a-z0-9')"
  psql -h "$PRIMARY_HOST" -U "$PGUSER" -d "$PGDB" -v ON_ERROR_STOP=1 <<EOF
SELECT pg_create_slot('${SLOT_NAME}'); -- placeholder, real fn below
EOF
  psql -h "$PRIMARY_HOST" -U "$PGUSER" -d "$PGDB" -v ON_ERROR_STOP=1 <<EOF
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_replication_slots WHERE slot_name = '${SLOT_NAME}') THEN
    PERFORM pg_create_logical_replication_slot('${SLOT_NAME}', 'pgoutput');
  END IF;
END\$\$;
EOF
done

echo "==> Replication slots ready."
psql -h "$PRIMARY_HOST" -U "$PGUSER" -d "$PGDB" -c "SELECT slot_name, plugin, slot_type, active FROM pg_replication_slots;"
