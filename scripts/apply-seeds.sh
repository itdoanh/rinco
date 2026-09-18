#!/bin/bash
# Apply all seed files in order
set +e
cd /tmp/seed
for f in $(ls *.sql | grep -v 00_master | sort); do
  echo "=== Applying seed: $f ==="
  psql -U rinco -d rinco -v ON_ERROR_STOP=0 -f "$f" 2>&1 | tail -3
done

# Apply expansion seeds
cd /tmp/seed/expansion
for f in $(ls *.sql | sort); do
  echo "=== Applying expansion: $f ==="
  psql -U rinco -d rinco -v ON_ERROR_STOP=0 -f "$f" 2>&1 | tail -3
done

echo "=== All seeds applied ==="
