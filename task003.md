## Goal
Allow users to upload and attach images/files in chat messages.

## Features
- [ ] Model: Media (id, message_id, url, mime, size)
- [ ] API: POST /media/presign (return upload URL)
- [ ] Local FS upload (dev) or S3-compatible
- [ ] Validate file type and size
- [ ] Link uploaded media to message
- [ ] Migration: create_media_table.sql
- [ ] Unit tests for upload handler
