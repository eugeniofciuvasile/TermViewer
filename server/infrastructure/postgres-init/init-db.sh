#!/bin/sh
set -eu

app_db="${POSTGRES_APP_DB:-termviewer}"

psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER}" --dbname "${POSTGRES_DB}" <<-EOSQL
SELECT format('CREATE DATABASE %I', '${app_db}')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${app_db}')\gexec
EOSQL
