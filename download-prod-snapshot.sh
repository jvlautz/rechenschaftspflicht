#!/usr/bin/env bash

set -euo pipefail

set -a
. <(grep '^LITESTREAM' ~/repos/rknt-server/.env)
set +a

rm -f ./src/data/*

LITESTREAM_BUCKET=rechenschaftspflicht litestream restore -config ./src/etc/litestream.yml -o ./src/data/state.db /app/data/state.db
