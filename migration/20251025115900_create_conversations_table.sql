-- Migration: Create conversations table
-- Created: 2025-10-25 11:59:00

-- +migrate Up
CREATE TABLE IF NOT EXISTS conversations (
    id SERIAL PRIMARY KEY,
    type VARCHAR(20) NOT NULL CHECK (type IN ('direct', 'group')),
    name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create index on type for faster lookups
CREATE INDEX IF NOT EXISTS idx_conversations_type ON conversations(type);

-- Create index on deleted_at for soft deletes
CREATE INDEX IF NOT EXISTS idx_conversations_deleted_at ON conversations(deleted_at);

-- +migrate Down
DROP INDEX IF EXISTS idx_conversations_deleted_at;
DROP INDEX IF EXISTS idx_conversations_type;
DROP TABLE IF EXISTS conversations;