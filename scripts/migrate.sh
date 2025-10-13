#!/bin/sh
set -e

# Wait for postgres to be ready
echo "Waiting for PostgreSQL to be ready..."
until PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c '\q' 2>/dev/null; do
  echo "Postgres is unavailable - sleeping"
  sleep 1
done

echo "Postgres is up - executing migrations"

# Run migrations using simple SQL files (only Up migrations)
for migration in /root/migrations/*.sql; do
  if [ -f "$migration" ]; then
    echo "Running migration: $(basename $migration)"
    # Extract only the "Up" section from migration files
    awk '/-- \+migrate Up/,/-- \+migrate Down/ { if ($0 !~ /-- \+migrate Down/) print }' "$migration" | \
    PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME"
  fi
done

echo "Migrations complete"
