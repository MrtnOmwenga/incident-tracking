#!/bin/bash
# start-services.sh

# Run migrations
docker compose --profile migrate up migration -d

# Wait for migrations to complete
until docker compose --profile migrate logs migration | grep 'Migrating done'; do
  echo "Waiting for migrations to complete..."
  sleep 5
done

# Start the rest of the services
docker compose up