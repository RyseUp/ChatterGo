# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:3.18

# Install required packages
RUN apk --no-cache add ca-certificates postgresql15-client 2>/dev/null || \
    apk --no-cache add ca-certificates 2>/dev/null || true

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

# Copy migration script (optional)
COPY scripts/migrate.sh ./migrate.sh 2>/dev/null || true
RUN chmod +x ./migrate.sh 2>/dev/null || true

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
