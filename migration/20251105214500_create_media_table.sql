-- Migration: create_media_table
-- Created: 2025-11-05 21:45:00 UTC

-- +migrate Up
CREATE TABLE IF NOT EXISTS media (
    id SERIAL PRIMARY KEY,
    message_id INTEGER NOT NULL,
    url VARCHAR(500) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL,
    filename VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_media_message FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_media_message_id ON media(message_id);
CREATE INDEX IF NOT EXISTS idx_media_deleted_at ON media(deleted_at);
CREATE INDEX IF NOT EXISTS idx_media_mime_type ON media(mime_type);

-- +migrate Down
DROP INDEX IF EXISTS idx_media_mime_type;
DROP INDEX IF EXISTS idx_media_deleted_at;
DROP INDEX IF EXISTS idx_media_message_id;
DROP TABLE IF EXISTS media;
