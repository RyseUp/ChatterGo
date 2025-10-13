# ChatterGo

ChatterGo is a real-time messaging platform designed to connect users instantly and securely — perfect for team collaboration, community chat, or customer support integration.

## Features

- 🔐 **Secure Authentication**: JWT-based authentication with access and refresh tokens
- 💬 **Real-time Messaging**: WebSocket-based instant messaging
- 🏢 **Room/Channel Support**: Create and join chat rooms
- 👥 **User Presence**: Track online/offline status of users
- 📜 **Message History**: Retrieve historical messages with pagination
- 🏗️ **Clean Architecture**: Separation of concerns for maintainability and scalability
- 🐳 **Docker Support**: Easy deployment with Docker and Docker Compose
- 🗄️ **PostgreSQL**: Reliable and scalable database backend

## Tech Stack

- **Backend Framework**: Go with Gin
- **Real-time Communication**: WebSocket (gorilla/websocket)
- **Database**: PostgreSQL
- **Authentication**: JWT (golang-jwt/jwt)
- **Password Hashing**: bcrypt
- **Containerization**: Docker & Docker Compose

## Architecture

ChatterGo follows Clean Architecture principles:

```
cmd/
  server/           # Application entry point
internal/
  domain/           # Business entities and DTOs
  repository/       # Data access interfaces and implementations
    postgres/       # PostgreSQL implementations
  usecase/          # Business logic layer
  delivery/         # Delivery mechanisms
    http/           # REST API handlers
    websocket/      # WebSocket handlers
pkg/
  config/           # Configuration management
  database/         # Database connection
  jwt/              # JWT token management
  middleware/       # HTTP middlewares
migrations/         # Database migrations
```

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 15 or higher
- Docker & Docker Compose (for containerized deployment)
- Make (optional, for using Makefile commands)

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/RyseUp/ChatterGo.git
cd ChatterGo
```

2. Start the application:
```bash
make docker-up
```

This will:
- Start PostgreSQL container
- Build and start the backend container
- Run database migrations automatically
- Expose the API on `http://localhost:8080`

3. Check the health:
```bash
curl http://localhost:8080/health
```

### Local Development

1. Start PostgreSQL:
```bash
docker run -d \
  --name chattergo-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=chattergo \
  -p 5432:5432 \
  postgres:15-alpine
```

2. Install golang-migrate:
```bash
make migrate
```

3. Run migrations:
```bash
make migrate-up
```

4. Set environment variables (optional, defaults will be used):
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=chattergo
export JWT_ACCESS_SECRET=your-secret-key
export JWT_REFRESH_SECRET=your-refresh-secret
```

5. Run the application:
```bash
make run
```

Or build and run:
```bash
make build
./bin/chattergo
```

## API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Authentication Endpoints

#### Register a new user
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "securepassword123"
}
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com",
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T10:00:00Z"
  }
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "securepassword123"
}
```

#### Refresh Token
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Room Endpoints (Protected)

All room endpoints require authentication. Include the access token in the Authorization header:
```
Authorization: Bearer <access_token>
```

#### Create a room
```http
POST /api/v1/rooms
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "General Discussion",
  "description": "A place for general chat"
}
```

#### List all rooms
```http
GET /api/v1/rooms?limit=20&offset=0
Authorization: Bearer <access_token>
```

#### Get room details
```http
GET /api/v1/rooms/:id
Authorization: Bearer <access_token>
```

#### Join a room
```http
POST /api/v1/rooms/join
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "room_id": 1
}
```

#### Leave a room
```http
DELETE /api/v1/rooms/:id/leave
Authorization: Bearer <access_token>
```

#### Get room members
```http
GET /api/v1/rooms/:id/members
Authorization: Bearer <access_token>
```

### Message Endpoints (Protected)

#### Send a message (REST)
```http
POST /api/v1/messages
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "room_id": 1,
  "content": "Hello, everyone!"
}
```

#### Get message history
```http
GET /api/v1/messages/history?room_id=1&limit=50&offset=0
Authorization: Bearer <access_token>
```

### WebSocket Connection (Protected)

Connect to WebSocket for real-time messaging:

```
ws://localhost:8080/api/v1/ws?room_id=1
Authorization: Bearer <access_token>
```

#### Send a message via WebSocket
```json
{
  "content": "Hello via WebSocket!"
}
```

#### Receive messages
Messages are broadcast to all connected clients in the room:
```json
{
  "id": 123,
  "room_id": 1,
  "user_id": 1,
  "username": "johndoe",
  "content": "Hello via WebSocket!",
  "created_at": "2024-01-01T10:30:00Z"
}
```

## Makefile Commands

```bash
make help           # Display help information
make run            # Run the application locally
make build          # Build the application binary
make test           # Run tests
make migrate        # Install golang-migrate tool
make migrate-up     # Run database migrations up
make migrate-down   # Run database migrations down
make docker-build   # Build docker images
make docker-up      # Start docker containers
make docker-down    # Stop docker containers
make docker-logs    # Show docker logs
make clean          # Clean build artifacts
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_HOST` | Server host | `0.0.0.0` |
| `SERVER_PORT` | Server port | `8080` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | PostgreSQL user | `postgres` |
| `DB_PASSWORD` | PostgreSQL password | `postgres` |
| `DB_NAME` | Database name | `chattergo` |
| `DB_SSLMODE` | SSL mode | `disable` |
| `JWT_ACCESS_SECRET` | JWT access token secret | `your-access-secret-key` |
| `JWT_REFRESH_SECRET` | JWT refresh token secret | `your-refresh-secret-key` |
| `JWT_ACCESS_EXPIRY_MIN` | Access token expiry (minutes) | `15` |
| `JWT_REFRESH_EXPIRY_MIN` | Refresh token expiry (minutes) | `10080` (7 days) |

⚠️ **Security Warning**: Always change the default JWT secrets in production!

## Testing

Run unit tests:
```bash
make test
```

Run tests with coverage:
```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Database Schema

### Users Table
- `id`: Primary key
- `username`: Unique username
- `email`: Unique email
- `password`: Hashed password
- `created_at`, `updated_at`: Timestamps

### Rooms Table
- `id`: Primary key
- `name`: Room name
- `description`: Room description
- `created_by`: User ID of creator
- `created_at`, `updated_at`: Timestamps

### Room Members Table
- `id`: Primary key
- `room_id`: Foreign key to rooms
- `user_id`: Foreign key to users
- `joined_at`: Join timestamp
- `is_online`: Online status
- `last_seen`: Last seen timestamp
- Unique constraint on (room_id, user_id)

### Messages Table
- `id`: Primary key
- `room_id`: Foreign key to rooms
- `user_id`: Foreign key to users
- `content`: Message content
- `created_at`: Timestamp

## Scalability

ChatterGo is designed with scalability in mind:

- **Horizontal Scaling**: Stateless API design allows multiple backend instances
- **WebSocket Hub**: In-memory hub can be replaced with Redis Pub/Sub for multi-instance deployments
- **Database Connection Pooling**: Efficient connection management
- **Indexed Queries**: Database indexes on frequently queried columns
- **Pagination Support**: All list endpoints support pagination

### Future Enhancements

- [ ] Redis integration for distributed WebSocket hub
- [ ] Message read receipts
- [ ] Typing indicators
- [ ] File upload support
- [ ] Push notifications
- [ ] Private direct messages
- [ ] User blocking/muting
- [ ] Message reactions
- [ ] Admin roles and permissions
- [ ] Rate limiting
- [ ] Prometheus metrics
- [ ] Kubernetes deployment manifests

## Deployment

### Production Considerations

1. **Environment Variables**: Set secure JWT secrets
2. **Database**: Use managed PostgreSQL service (AWS RDS, GCP Cloud SQL, etc.)
3. **HTTPS**: Deploy behind a reverse proxy (Nginx, Traefik) with SSL
4. **CORS**: Configure CORS for your frontend domain
5. **Monitoring**: Add logging, metrics, and alerting
6. **Backups**: Regular database backups
7. **Rate Limiting**: Implement rate limiting for API endpoints

### Docker Production Deployment

1. Build the image:
```bash
docker build -t chattergo:latest .
```

2. Run with production settings:
```bash
docker run -d \
  --name chattergo \
  -p 8080:8080 \
  -e JWT_ACCESS_SECRET=your-production-secret \
  -e JWT_REFRESH_SECRET=your-production-refresh-secret \
  -e DB_HOST=your-db-host \
  -e DB_PASSWORD=your-db-password \
  chattergo:latest
```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues, questions, or contributions, please open an issue on GitHub.

## Acknowledgments

- Built with [Gin](https://github.com/gin-gonic/gin) web framework
- Real-time communication powered by [Gorilla WebSocket](https://github.com/gorilla/websocket)
- Clean Architecture principles by Robert C. Martin
