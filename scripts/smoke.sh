#!/bin/sh
set -eu
go run ./cmd/manifest-simulator -url "${HAZMAT_URL:-http://127.0.0.1:32710}" -token "${HAZMAT_ADMIN_TOKEN:-hazmat-admin-token}"
