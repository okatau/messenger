#!/bin/sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

for f in $(ls "$SCRIPT_DIR/migrations"/*.up.sql | sort); do
  echo "Applying $f"
  docker exec -i pg-prod psql -U postgres -d messenger < "$f"
done

echo "All migrations applied"
