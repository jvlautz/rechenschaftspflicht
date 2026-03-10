#!/bin/sh
set -e

if [ -f /app/data/state.db ]; then
    echo "Database already exists, skipping restore"
else
    echo "No database found, attempting restore from replica"
    litestream restore -if-db-not-exists -if-replica-exists /app/data/state.db || true
fi

exec litestream replicate
