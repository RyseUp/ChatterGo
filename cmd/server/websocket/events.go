package websocket

const (
	EventConnect    = "connect"
	EventDisconnect = "disconnect"

	EventJoinRoom  = "join_room"
	EventLeaveRoom = "leave_room"

	EventTypingStart = "typing.start"
	EventTypingStop  = "typing.stop"

	EventMessageSend    = "message.send"    // client -> server
	EventMessageCreated = "message.created" // server -> room

	EventDeliveryReceipt = "delivery.receipt" // client -> server
	EventDeliveryUpdated = "delivery.updated" // server -> room

	EventPresenceOnline  = "presence.online" // server -> room or user followers
	EventPresenceOffline = "presence.offline"

	EventResume    = "resume" // client -> server (resume_token)
	EventServerAck = "server.ack"
	EventError     = "error"

	EventPing = "ping" // client -> server (heartbeat)
	EventPong = "pong" // server -> client (heartbeat response)
)
