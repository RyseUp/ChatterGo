# Task 003 – WebSocket Realtime Service

## Goal

Provide real-time delivery of messages and user presence.

## Features

- [x] Setup `/ws` WebSocket endpoint (integrated via `/socket.io/*`)
- [x] Authenticate socket with JWT
- [x] Track user online/offline status
- [x] Join/leave conversation rooms
- [x] Broadcast message.created events (with DB persistence)
- [x] Broadcast typing.start/stop
- [x] Send delivery receipts (delivered/read) - ready for DB schema extension
- [x] Implement ping/pong heartbeat
- [x] Graceful reconnect with resume token
