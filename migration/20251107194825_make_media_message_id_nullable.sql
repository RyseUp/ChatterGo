-- Migration: make_media_message_id_nullable
-- Created: 2025-11-07 19:48:25 UTC

-- +migrate Up
-- Drop the foreign key constraint first
ALTER TABLE media DROP CONSTRAINT IF EXISTS fk_media_message;

-- Make message_id nullable
ALTER TABLE media ALTER COLUMN message_id DROP NOT NULL;

-- Re-add the foreign key constraint (allowing NULL)
ALTER TABLE media ADD CONSTRAINT fk_media_message 
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE;

-- +migrate Down
-- Drop the foreign key constraint
ALTER TABLE media DROP CONSTRAINT IF EXISTS fk_media_message;

-- Make message_id NOT NULL again
ALTER TABLE media ALTER COLUMN message_id SET NOT NULL;

-- Re-add the foreign key constraint
ALTER TABLE media ADD CONSTRAINT fk_media_message 
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE;
