#!/bin/sh
set -e

echo "Waiting for database to be ready..."
# Wait for MySQL to be ready (with timeout)
max_attempts=30
attempt=0
until nc -z mysql 3306 || [ $attempt -eq $max_attempts ]; do
  attempt=$((attempt + 1))
  echo "Waiting for MySQL... (attempt $attempt/$max_attempts)"
  sleep 2
done

if [ $attempt -eq $max_attempts ]; then
  echo "ERROR: MySQL did not become ready in time"
  exit 1
fi

echo "Waiting for Redis to be ready..."
# Wait for Redis to be ready (with timeout)
attempt=0
until nc -z redis 6379 || [ $attempt -eq $max_attempts ]; do
  attempt=$((attempt + 1))
  echo "Waiting for Redis... (attempt $attempt/$max_attempts)"
  sleep 2
done

if [ $attempt -eq $max_attempts ]; then
  echo "ERROR: Redis did not become ready in time"
  exit 1
fi

echo "Database and Redis are ready!"
echo "Running seed script..."
/app/seed

if [ $? -ne 0 ]; then
  echo "WARNING: Seed script failed, but continuing with server startup..."
fi

echo "Starting server..."
exec /app/server

