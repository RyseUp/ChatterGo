# Contributing to ChatterGo

Thank you for your interest in contributing to ChatterGo! This document provides guidelines and instructions for contributing.

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and encourage diverse perspectives
- Focus on constructive feedback

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Create a new branch for your feature or bugfix
4. Make your changes
5. Test your changes
6. Submit a pull request

## Development Setup

```bash
# Clone the repository
git clone https://github.com/RyseUp/ChatterGo.git
cd ChatterGo

# Install dependencies
go mod download

# Start PostgreSQL
docker run -d --name chattergo-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=chattergo \
  -p 5432:5432 \
  postgres:15-alpine

# Run migrations
make migrate-up

# Run the application
make run
```

## Project Structure

```
ChatterGo/
├── cmd/              # Application entry points
├── internal/         # Private application code
│   ├── domain/       # Business entities and DTOs
│   ├── repository/   # Data access interfaces and implementations
│   ├── usecase/      # Business logic
│   └── delivery/     # HTTP and WebSocket handlers
├── pkg/              # Public libraries
├── migrations/       # Database migrations
└── examples/         # Example code and clients
```

## Coding Standards

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` to format code
- Use `golint` for linting
- Write meaningful variable and function names
- Add comments for exported functions and types

### Clean Architecture

ChatterGo follows Clean Architecture principles:

1. **Domain Layer** (`internal/domain`): Business entities, no external dependencies
2. **Repository Layer** (`internal/repository`): Data access interfaces and implementations
3. **Use Case Layer** (`internal/usecase`): Business logic, orchestrates repositories
4. **Delivery Layer** (`internal/delivery`): HTTP handlers, WebSocket handlers

### Dependency Rule

Dependencies should always point inward:
- Delivery → Use Case → Repository → Domain
- Domain has no dependencies
- Use Cases depend only on Repository interfaces and Domain
- Repositories depend only on Domain

## Adding New Features

### Adding a New Endpoint

1. Define request/response DTOs in `internal/domain/`
2. Add repository interface method if needed in `internal/repository/`
3. Implement repository method in `internal/repository/postgres/`
4. Add use case method in `internal/usecase/`
5. Create handler in `internal/delivery/http/`
6. Register route in `cmd/server/main.go`

Example:

```go
// 1. Domain (internal/domain/example.go)
type ExampleRequest struct {
    Name string `json:"name" binding:"required"`
}

// 2. Repository Interface (internal/repository/example_repository.go)
type ExampleRepository interface {
    Create(ctx context.Context, example *domain.Example) error
}

// 3. Repository Implementation (internal/repository/postgres/example_repository.go)
func (r *exampleRepository) Create(ctx context.Context, example *domain.Example) error {
    // Implementation
}

// 4. Use Case (internal/usecase/example_usecase.go)
func (uc *exampleUseCase) CreateExample(ctx context.Context, req *domain.ExampleRequest) error {
    // Business logic
}

// 5. Handler (internal/delivery/http/example_handler.go)
func (h *ExampleHandler) Create(c *gin.Context) {
    // Handle HTTP request
}
```

## Testing

### Unit Tests

Write unit tests for:
- Domain logic
- Use cases (with mocked repositories)
- Utilities

```bash
go test ./...
```

### Integration Tests

Integration tests should:
- Use a test database
- Test complete flows
- Clean up after themselves

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Database Migrations

### Adding a Migration

1. Create a new SQL file in `migrations/` directory
2. Name it with incrementing number: `00X_description.sql`
3. Include both Up and Down migrations:

```sql
-- +migrate Up
CREATE TABLE example (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

-- +migrate Down
DROP TABLE example;
```

4. Update `migrations/init.sql` if needed

## Pull Request Process

1. **Branch Naming**: Use descriptive names
   - `feature/add-user-profile`
   - `bugfix/fix-websocket-disconnect`
   - `docs/update-readme`

2. **Commit Messages**: Write clear, descriptive commit messages
   - Use present tense ("Add feature" not "Added feature")
   - Reference issues when applicable (#123)

3. **Testing**: Ensure all tests pass
   ```bash
   make test
   ```

4. **Documentation**: Update README if needed

5. **Code Review**: Be responsive to feedback

## Common Tasks

### Adding a New Database Table

1. Create migration file
2. Add entity to `internal/domain/`
3. Create repository interface
4. Implement PostgreSQL repository
5. Add use case methods
6. Create HTTP handlers

### Adding Authentication to an Endpoint

```go
protected := router.Group("/api/v1")
protected.Use(middleware.AuthMiddleware(tokenManager))
{
    protected.GET("/protected", handler.ProtectedEndpoint)
}
```

### Accessing User from Context

```go
func (h *Handler) MyHandler(c *gin.Context) {
    userID := middleware.GetUserID(c)
    username := middleware.GetUsername(c)
    // Use user info
}
```

## Questions or Problems?

- Open an issue on GitHub
- Check existing issues and pull requests
- Review the README and documentation

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (MIT License).
