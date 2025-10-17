# ChatterGo (Backend)

A real-time messaging backend focused on instant, reliable communication. Phase 1 delivers:

- WebSocket gateway (native) with broadcast hub
- REST healthcheck
- PostgreSQL + GORM models (User, Room, Message)
- Repository/Service layering
- Config via YAML + env overrides
- Migrator entrypoint

## Quick start

1. Set DB in `config/default.yaml` (or env vars)
2. `make migrate`
3. `make run`
4. Connect your WS client to `ws://localhost:8080/ws?user_id=<id>`

Tech specifics live in the repo code and future README sections.
