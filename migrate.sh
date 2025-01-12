#!/bin/sh
set -e

# Print environment variables for debugging (optional)
echo "DB_USER: $DB_USER"
echo "DB_NAME: $DB_NAME"
echo "DB_HOST: $DB_HOST"

CONNECTION_STRING="postgres://$DB_USER:$DB_PASSWORD@postgres:5432/$DB_NAME?sslmode=disable"
echo "Using connection string: $CONNECTION_STRING"

migrate -verbose -source "file:///migrations" -database "$CONNECTION_STRING" up