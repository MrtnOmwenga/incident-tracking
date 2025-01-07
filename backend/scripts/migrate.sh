#!/bin/bash

# Variables
MIGRATIONS_PATH="./migrations"
DATABASE_URL="postgres://admin:password@localhost:5432/incident-tracking?sslmode=disable"

# Run the migrate command
docker run --rm -v $(pwd)/migrations:/migrations --network="host" migrate/migrate \
  -path=/migrations \
  -database "$DATABASE_URL" up
