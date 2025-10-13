package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	httpHandler "github.com/RyseUp/ChatterGo/internal/delivery/http"
	"github.com/RyseUp/ChatterGo/internal/delivery/websocket"
	"github.com/RyseUp/ChatterGo/internal/repository/postgres"
	"github.com/RyseUp/ChatterGo/internal/usecase"
	"github.com/RyseUp/ChatterGo/pkg/config"
	"github.com/RyseUp/ChatterGo/pkg/database"
	"github.com/RyseUp/ChatterGo/pkg/jwt"
	"github.com/RyseUp/ChatterGo/pkg/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to database successfully")

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	messageRepo := postgres.NewMessageRepository(db)
	roomRepo := postgres.NewRoomRepository(db)

	// Initialize JWT token manager
	tokenManager := jwt.NewTokenManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.GetAccessExpiry(),
		cfg.JWT.GetRefreshExpiry(),
	)

	// Initialize use cases
	authUseCase := usecase.NewAuthUseCase(userRepo, tokenManager)
	messageUseCase := usecase.NewMessageUseCase(messageRepo, roomRepo, userRepo)
	roomUseCase := usecase.NewRoomUseCase(roomRepo)

	// Initialize WebSocket hub
	hub := websocket.NewHub(roomRepo)
	go hub.Run()

	// Initialize handlers
	authHandler := httpHandler.NewAuthHandler(authUseCase)
	messageHandler := httpHandler.NewMessageHandler(messageUseCase)
	roomHandler := httpHandler.NewRoomHandler(roomUseCase)
	wsHandler := httpHandler.NewWebSocketHandler(hub)

	// Setup router
	router := setupRouter(authHandler, messageHandler, roomHandler, wsHandler, tokenManager)

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:    cfg.Server.Host + ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited successfully")
}

func setupRouter(
	authHandler *httpHandler.AuthHandler,
	messageHandler *httpHandler.MessageHandler,
	roomHandler *httpHandler.RoomHandler,
	wsHandler *httpHandler.WebSocketHandler,
	tokenManager *jwt.TokenManager,
) *gin.Engine {
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(tokenManager))
		{
			// Room routes
			rooms := protected.Group("/rooms")
			{
				rooms.POST("", roomHandler.CreateRoom)
				rooms.GET("", roomHandler.ListRooms)
				rooms.GET("/:id", roomHandler.GetRoom)
				rooms.POST("/join", roomHandler.JoinRoom)
				rooms.DELETE("/:id/leave", roomHandler.LeaveRoom)
				rooms.GET("/:id/members", roomHandler.GetRoomMembers)
			}

			// Message routes
			messages := protected.Group("/messages")
			{
				messages.POST("", messageHandler.SendMessage)
				messages.GET("/history", messageHandler.GetMessageHistory)
			}

			// WebSocket route
			protected.GET("/ws", wsHandler.HandleWebSocket)
		}
	}

	return router
}
