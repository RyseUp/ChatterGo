-- Migration: Create conversation_members table
-- Created: 2025-10-25 11:59:01

-- +migrate Up
CREATE TABLE IF NOT EXISTS conversation_members (
    id SERIAL PRIMARY KEY,
    conversation_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(conversation_id, user_id)
);

-- Add foreign key constraints
ALTER TABLE conversation_members 
ADD CONSTRAINT fk_conversation_members_conversation_id 
FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE;

ALTER TABLE conversation_members 
ADD CONSTRAINT fk_conversation_members_user_id 
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_conversation_members_conversation_id ON conversation_members(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_members_user_id ON conversation_members(user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_members_role ON conversation_members(role);

-- +migrate Down
DROP INDEX IF EXISTS idx_conversation_members_role;
DROP INDEX IF EXISTS idx_conversation_members_user_id;
DROP INDEX IF EXISTS idx_conversation_members_conversation_id;
ALTER TABLE conversation_members DROP CONSTRAINT IF EXISTS fk_conversation_members_user_id;
ALTER TABLE conversation_members DROP CONSTRAINT IF EXISTS fk_conversation_members_conversation_id;
DROP TABLE IF EXISTS conversation_members;