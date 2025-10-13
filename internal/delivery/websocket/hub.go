package websocket

import (
	"context"
	"log"

	"github.com/RyseUp/ChatterGo/internal/repository"
)

type BroadcastMessage struct {
	roomID  int64
	message []byte
}

type Hub struct {
	// Registered clients per room
	rooms map[int64]map[*Client]bool

	// Inbound messages from the clients
	broadcast chan *BroadcastMessage

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Room repository for updating member status
	roomRepo repository.RoomRepository
}

func NewHub(roomRepo repository.RoomRepository) *Hub {
	return &Hub{
		rooms:      make(map[int64]map[*Client]bool),
		broadcast:  make(chan *BroadcastMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		roomRepo:   roomRepo,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			// Create room if it doesn't exist
			if h.rooms[client.roomID] == nil {
				h.rooms[client.roomID] = make(map[*Client]bool)
			}
			h.rooms[client.roomID][client] = true

			// Update member status to online
			ctx := context.Background()
			if err := h.roomRepo.UpdateMemberStatus(ctx, client.roomID, client.userID, true); err != nil {
				log.Printf("error updating member status: %v", err)
			}

		case client := <-h.unregister:
			if clients, ok := h.rooms[client.roomID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)

					// Update member status to offline
					ctx := context.Background()
					if err := h.roomRepo.UpdateMemberStatus(ctx, client.roomID, client.userID, false); err != nil {
						log.Printf("error updating member status: %v", err)
					}

					// Remove room if empty
					if len(clients) == 0 {
						delete(h.rooms, client.roomID)
					}
				}
			}

		case msg := <-h.broadcast:
			// Broadcast to all clients in the room
			if clients, ok := h.rooms[msg.roomID]; ok {
				for client := range clients {
					select {
					case client.send <- msg.message:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
		}
	}
}

func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}
