# ChatterGo Architecture

## Overview

ChatterGo is a real-time chat backend built with Go, following Clean Architecture principles. It provides RESTful APIs for authentication and message management, along with WebSocket support for real-time communication.

## Technology Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PostgreSQL 15
- **Real-time**: WebSocket (gorilla/websocket)
- **Authentication**: JWT (golang-jwt/jwt)
- **Password Hashing**: bcrypt
- **Containerization**: Docker & Docker Compose

## Architecture Layers

### 1. Domain Layer (`internal/domain/`)

The innermost layer containing business entities and data transfer objects (DTOs).

**Entities:**
- `User`: User account information
- `Room`: Chat room/channel
- `Message`: Chat message
- `RoomMember`: Room membership and presence

**DTOs:**
- `RegisterRequest`, `LoginRequest`, `AuthResponse`
- `SendMessageRequest`, `MessageHistoryRequest`
- `CreateRoomRequest`, `JoinRoomRequest`

**Dependencies:** None (pure business logic)

### 2. Repository Layer (`internal/repository/`)

Defines interfaces for data access and provides PostgreSQL implementations.

**Interfaces:**
- `UserRepository`: User CRUD operations
- `MessageRepository`: Message storage and retrieval
- `RoomRepository`: Room management and membership

**Implementations:** (`internal/repository/postgres/`)
- PostgreSQL-based implementations using `database/sql`
- Efficient queries with proper indexing
- Connection pooling

**Dependencies:** Domain layer only

### 3. Use Case Layer (`internal/usecase/`)

Contains business logic and orchestrates operations between repositories.

**Use Cases:**
- `AuthUseCase`: User registration, login, token refresh
- `MessageUseCase`: Send messages, retrieve history
- `RoomUseCase`: Create rooms, manage membership

**Business Rules:**
- Password hashing and verification
- Token generation and validation
- Authorization checks (room membership)
- Data validation

**Dependencies:** Repository interfaces, Domain layer

### 4. Delivery Layer (`internal/delivery/`)

Handles external communication via HTTP and WebSocket.

**HTTP Handlers** (`internal/delivery/http/`)
- `AuthHandler`: Authentication endpoints
- `MessageHandler`: Message REST API
- `RoomHandler`: Room management API
- `WebSocketHandler`: WebSocket connection upgrade

**WebSocket** (`internal/delivery/websocket/`)
- `Hub`: Central message broker
- `Client`: Per-connection handler
- Real-time message broadcasting
- Presence tracking

**Dependencies:** Use case layer, Domain layer

## Component Diagram

```
┌─────────────────────────────────────────────────────────┐
│                     Clients                              │
│  (Web App, Mobile App, API Consumers)                   │
└────────────────┬────────────────────────────────────────┘
                 │
        ┌────────┴─────────┐
        │                  │
        ▼                  ▼
┌──────────────┐   ┌──────────────┐
│  REST API    │   │  WebSocket   │
│  (Gin)       │   │  (gorilla)   │
└──────┬───────┘   └──────┬───────┘
       │                  │
       │  Delivery Layer  │
       └──────────┬────────┘
                  │
       ┌──────────┴──────────┐
       │   Use Case Layer    │
       │  (Business Logic)   │
       └──────────┬──────────┘
                  │
       ┌──────────┴──────────┐
       │  Repository Layer   │
       │  (Data Access)      │
       └──────────┬──────────┘
                  │
       ┌──────────┴──────────┐
       │    PostgreSQL       │
       └─────────────────────┘
```

## Data Flow

### Authentication Flow

```
Client → POST /api/v1/auth/register
  → AuthHandler.Register()
  → AuthUseCase.Register()
  → UserRepository.Create()
  → PostgreSQL
  ← User created
  ← Tokens generated (JWT)
  ← AuthResponse
```

### Real-time Messaging Flow

```
Client → WebSocket Connect /api/v1/ws?room_id=1
  → WebSocketHandler.HandleWebSocket()
  → Create Client
  → Register with Hub
  
Client → Send Message (JSON)
  → Client.ReadPump()
  → Hub.Broadcast()
  → All Clients in Room
  ← Receive Message
```

### Message History Flow

```
Client → GET /api/v1/messages/history?room_id=1
  → JWT Middleware (validate token)
  → MessageHandler.GetMessageHistory()
  → MessageUseCase.GetMessageHistory()
  → Check room membership
  → MessageRepository.GetByRoomID()
  → PostgreSQL (with JOIN on users)
  ← Messages with usernames
```

## Database Schema

### Tables

1. **users**
   - Primary key: `id`
   - Unique: `email`, `username`
   - Indexes: `email`, `username`

2. **rooms**
   - Primary key: `id`
   - Foreign key: `created_by` → `users(id)`
   - Index: `created_by`

3. **room_members**
   - Primary key: `id`
   - Foreign keys: `room_id` → `rooms(id)`, `user_id` → `users(id)`
   - Unique: `(room_id, user_id)`
   - Indexes: `room_id`, `user_id`
   - Tracks: Online status, last seen

4. **messages**
   - Primary key: `id`
   - Foreign keys: `room_id` → `rooms(id)`, `user_id` → `users(id)`
   - Indexes: `room_id`, `user_id`, `created_at`

### Relationships

```
users (1) ─── creates ──→ (N) rooms
users (N) ─── member of ──→ (N) rooms (through room_members)
users (1) ─── sends ──→ (N) messages
rooms (1) ─── contains ──→ (N) messages
```

## Security

### Authentication
- JWT-based with access and refresh tokens
- Access token: Short-lived (15 minutes default)
- Refresh token: Long-lived (7 days default)
- Tokens signed with HMAC-SHA256

### Authorization
- Middleware validates JWT on protected routes
- User context extracted from token
- Room membership checked before operations
- WebSocket connections require valid token

### Password Security
- bcrypt hashing with default cost factor
- Passwords never returned in API responses
- Salting handled automatically by bcrypt

## Scalability Considerations

### Current Architecture
- Stateless API servers (horizontal scaling possible)
- In-memory WebSocket hub (single instance)
- PostgreSQL connection pooling
- Indexed database queries

### Scaling Strategies

1. **API Servers**: Multiple instances behind load balancer
2. **WebSocket Hub**: Replace with Redis Pub/Sub for multi-instance support
3. **Database**: Read replicas, connection pooling, query optimization
4. **Caching**: Add Redis for session management and frequent queries
5. **Message Queue**: Add for async operations and notifications

### Future Enhancements
- Redis integration for distributed WebSocket
- Message queue (RabbitMQ, Kafka) for notifications
- CDN for static assets
- Elasticsearch for message search
- Microservices separation (auth, messaging, presence)

## Configuration

Environment-based configuration via `pkg/config/`:
- Server settings (host, port)
- Database connection
- JWT secrets and expiry
- All configurable via environment variables

## Monitoring & Observability

### Current State
- Structured logging
- Error handling at each layer
- Health check endpoint

### Recommended Additions
- Prometheus metrics
- Distributed tracing (Jaeger, OpenTelemetry)
- Log aggregation (ELK stack)
- APM tools (New Relic, DataDog)

## Testing Strategy

### Unit Tests
- Domain logic validation
- Use case tests with mocked repositories
- JWT token generation/validation

### Integration Tests
- API endpoint testing
- Database integration
- WebSocket connection handling

### Load Tests
- Concurrent WebSocket connections
- Message throughput
- Database query performance

## Deployment

### Docker
- Multi-stage build for smaller images
- Alpine Linux base for minimal footprint
- Health checks for container orchestration

### Docker Compose
- Development environment setup
- PostgreSQL with persistent volumes
- Automatic migration on startup

### Production Considerations
- Use managed PostgreSQL (RDS, Cloud SQL)
- Configure CORS properly
- Enable HTTPS/TLS
- Set secure JWT secrets
- Implement rate limiting
- Configure log rotation
- Set up backups

## API Endpoints Summary

### Public Endpoints
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh access token

### Protected Endpoints (require JWT)
- `POST /api/v1/rooms` - Create room
- `GET /api/v1/rooms` - List rooms
- `GET /api/v1/rooms/:id` - Get room details
- `POST /api/v1/rooms/join` - Join room
- `DELETE /api/v1/rooms/:id/leave` - Leave room
- `GET /api/v1/rooms/:id/members` - Get room members
- `POST /api/v1/messages` - Send message
- `GET /api/v1/messages/history` - Get message history
- `GET /api/v1/ws` - WebSocket connection

## Development Workflow

1. **Local Development**: Use `make run` with local PostgreSQL
2. **Testing**: Run `make test` for unit tests
3. **Building**: Use `make build` for production binary
4. **Docker**: Use `make docker-up` for containerized environment
5. **Migrations**: Use `make migrate-up` to apply database changes

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed contribution guidelines.

## License

MIT License - See [LICENSE](LICENSE) file for details.
