#!/bin/sh
set -e

SRC="/shared/env/app.env"
DST="/app/.env"

if [ -f "$SRC" ]; then
  cp "$SRC" "$DST"
  chmod 0400 "$DST" || true
fi

exec "$@"
