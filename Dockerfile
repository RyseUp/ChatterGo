# --- build stage
FROM golang:1.23 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# --- runtime
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/server ./
COPY config ./config
ENV GIN_MODE=release
CMD ["/app/server"]