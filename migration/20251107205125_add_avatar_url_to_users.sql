-- Migration: add_avatar_url_to_users
-- Created: 2025-11-07 20:51:25 UTC

-- +migrate Up
-- Add avatar_url column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(500);

-- Create index for faster lookups (optional, but useful if querying by avatar_url)
CREATE INDEX IF NOT EXISTS idx_users_avatar_url ON users(avatar_url) WHERE avatar_url IS NOT NULL;

-- +migrate Down
-- Remove index
DROP INDEX IF EXISTS idx_users_avatar_url;

-- Remove avatar_url column
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
