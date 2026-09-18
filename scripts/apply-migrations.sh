#!/bin/bash
# Apply all SQL migrations
set +e
cd /tmp/sql
for f in $(ls *.sql | sort); do
  echo "=== Applying: $f ==="
  psql -U rinco -d rinco -v ON_ERROR_STOP=0 -f "$f" 2>&1 | tail -5
done
echo "=== All migrations applied ==="
