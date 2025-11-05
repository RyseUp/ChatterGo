# Media Upload Feature

This document describes the media upload functionality for the ChatterGo chat platform.

## Overview

The media upload feature allows users to attach images and files to their chat messages. It supports both local filesystem storage (for development) and can be extended for S3-compatible storage (for production).

## Features

- ✅ **Media Model**: Database model for storing media metadata
- ✅ **File Upload API**: RESTful endpoints for uploading files
- ✅ **Presigned URLs**: Generate secure upload URLs (placeholder for S3)
- ✅ **File Validation**: Size and MIME type validation
- ✅ **Local Storage**: Development-friendly local filesystem storage
- ✅ **Message Integration**: Link media to chat messages
- ✅ **Database Migration**: SQL migration for media table
- ✅ **Unit Tests**: Comprehensive test coverage

## API Endpoints

### 1. Presign Upload URL
```http
POST /api/v1/media/presign
Authorization: Bearer <token>
Content-Type: application/json

{
  "filename": "image.jpg",
  "mime_type": "image/jpeg",
  "size": 1048576
}
```

**Response:**
```json
{
  "upload_url": "http://localhost:9090/api/v1/media/upload?media_id=...",
  "media_id": "1699123456_abc123def.jpg",
  "expires_at": 1699124356
}
```

### 2. Upload File
```http
POST /api/v1/media/upload
Authorization: Bearer <token>
Content-Type: multipart/form-data

file: <binary_data>
message_id: 123 (optional)
```

**Response:**
```json
{
  "media_id": 1,
  "url": "http://localhost:9090/uploads/1699123456_abc123def.jpg",
  "filename": "image.jpg",
  "mime_type": "image/jpeg",
  "size": 1048576
}
```

### 3. Get Media
```http
GET /api/v1/media/{id}
Authorization: Bearer <token>
```

### 4. Delete Media
```http
DELETE /api/v1/media/{id}
Authorization: Bearer <token>
```

## Configuration

### Default Settings
```go
MaxFileSize: 10 * 1024 * 1024, // 10MB
AllowedMimeTypes: [
  "image/jpeg", "image/png", "image/gif", "image/webp",
  "application/pdf", "text/plain",
  "application/msword", 
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
],
StorageType: "local",
LocalStoragePath: "./uploads",
BaseURL: "http://localhost:9090"
```

### Environment Variables
You can override these settings using environment variables:
- `MEDIA_MAX_FILE_SIZE`: Maximum file size in bytes
- `MEDIA_STORAGE_TYPE`: "local" or "s3"
- `MEDIA_LOCAL_PATH`: Local storage directory
- `MEDIA_BASE_URL`: Base URL for serving files

## Database Schema

### Media Table
```sql
CREATE TABLE media (
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
```

### Indexes
- `idx_media_message_id`: Fast lookup by message
- `idx_media_deleted_at`: Soft delete support
- `idx_media_mime_type`: Filter by file type

## Usage Examples

### Frontend Integration

#### 1. Upload with Presigned URL
```javascript
// Step 1: Get presigned URL
const presignResponse = await fetch('/api/v1/media/presign', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    filename: file.name,
    mime_type: file.type,
    size: file.size
  })
});

const { upload_url } = await presignResponse.json();

// Step 2: Upload file
const formData = new FormData();
formData.append('file', file);

const uploadResponse = await fetch(upload_url, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`
  },
  body: formData
});
```

#### 2. Direct Upload
```javascript
const formData = new FormData();
formData.append('file', file);
formData.append('message_id', messageId);

const response = await fetch('/api/v1/media/upload', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`
  },
  body: formData
});
```

### Backend Integration

#### Linking Media to Messages
```go
// When creating a message with media
message := &models.Message{
    ConversationID: conversationID,
    SenderID: userID,
    Content: "Check out this image!",
}

// Create message first
err := repo.CreateMessage(ctx, message)

// Then create media record
media := &models.Media{
    MessageID: message.ID,
    URL: uploadedFileURL,
    MimeType: "image/jpeg",
    Size: fileSize,
    Filename: originalFilename,
}

err = repo.CreateMedia(ctx, media)
```

#### Retrieving Messages with Media
```go
// Get messages with preloaded media
messages, err := repo.GetMessagesByConversationID(ctx, conversationID, 50, 0)

// Media will be included in the response due to GORM relationships
for _, message := range messages {
    for _, media := range message.Media {
        fmt.Printf("Media: %s (%s)\n", media.Filename, media.MimeType)
    }
}
```

## File Storage

### Local Storage (Development)
- Files stored in `./uploads/` directory
- Served via Gin static file handler at `/uploads/*`
- Unique filenames generated with timestamp + random string

### S3 Storage (Production Ready)
The codebase is structured to easily add S3 support:

```go
// Future S3 implementation
func (s *ServiceServer) SaveFileToS3(file *multipart.FileHeader, config MediaConfig) (string, error) {
    // AWS S3 upload logic
    // Return S3 URL
}

func (s *ServiceServer) GenerateS3PresignedURL(filename string, config MediaConfig) (string, error) {
    // Generate actual S3 presigned URL
    // Return presigned URL with expiration
}
```

## Security Considerations

### File Validation
- **Size Limits**: Configurable maximum file size (default: 10MB)
- **MIME Type Checking**: Only allowed file types accepted
- **Content Validation**: Uses `http.DetectContentType()` to verify actual file content
- **Filename Sanitization**: Generates unique, safe filenames

### Access Control
- **Authentication Required**: All endpoints require valid JWT token
- **File Serving**: Static files served publicly (consider adding auth for sensitive files)
- **Database Constraints**: Foreign key constraints ensure data integrity

### Best Practices
1. **Virus Scanning**: Consider adding antivirus scanning for uploaded files
2. **CDN Integration**: Use CDN for better performance and caching
3. **Backup Strategy**: Implement backup for uploaded files
4. **Monitoring**: Track upload metrics and storage usage

## Testing

### Running Tests
```bash
# Run all media tests
go test ./internal/services -v -run TestMedia

# Run specific test
go test ./internal/services -v -run TestUploadFile

# Run with coverage
go test ./internal/services -cover -coverprofile=coverage.out
```

### Test Coverage
- ✅ Presign URL generation
- ✅ File upload validation
- ✅ MIME type checking
- ✅ File size validation
- ✅ Database operations
- ✅ Error handling
- ✅ File cleanup on errors

## Migration

### Running Migration
```bash
# Apply migration
go run ./cmd/migrator

# Or using make
make migrate
```

### Rollback
```bash
# Rollback last migration
go run ./cmd/migrator -down -steps 1
```

## Monitoring and Metrics

Consider tracking these metrics:
- Upload success/failure rates
- File size distribution
- Popular file types
- Storage usage over time
- Upload performance metrics

## Future Enhancements

1. **Image Processing**: Thumbnail generation, image resizing
2. **CDN Integration**: CloudFront, CloudFlare integration
3. **Advanced Validation**: File content scanning, metadata extraction
4. **Compression**: Automatic file compression for large files
5. **Chunked Upload**: Support for large file uploads
6. **Progress Tracking**: Real-time upload progress
7. **File Versioning**: Keep multiple versions of files
8. **Expiration**: Automatic cleanup of old files
