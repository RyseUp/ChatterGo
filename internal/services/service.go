package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/RyseUp/ChatterGo/cmd/server/websocket"
	"github.com/RyseUp/ChatterGo/config"
	"github.com/RyseUp/ChatterGo/internal/middleware"
	"github.com/RyseUp/ChatterGo/internal/repositories"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// WebSocketHub interface for notification broadcasting
type WebSocketHub interface {
	BroadcastToUser(userID uint, eventType string, data interface{}) error
	IsUserOnline(userID uint) bool
	GetUserSockets(userID uint) []string
}

type ServiceServer struct {
	cfg      *config.Config
	r        repositories.Repository
	svr      *http.Server
	wsServer *websocket.WebSocketServer
	wsHub    WebSocketHub
}

func NewServer(cfg *config.Config, r repositories.Repository) (*ServiceServer, error) {
	router := gin.Default()

	// Initialize WebSocket server with JWT secret
	wsServer := websocket.NewWebSocketServer(r, cfg.JWT.Secret)
	
	// Start WebSocket server in background
	go wsServer.Run()

	s := &ServiceServer{
		cfg:      cfg,
		r:        r,
		wsServer: wsServer,
		wsHub:    wsServer, // WebSocketServer implements WebSocketHub interface
		svr: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: router,
		},
	}

	s.setupRoutes(router)

	return s, nil
}

func (s *ServiceServer) setupRoutes(router *gin.Engine) {
	// CORS middleware - allow specific origins for development
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:8080", "http://127.0.0.1:5173", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "Access-Control-Request-Method", "Access-Control-Request-Headers"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
		AllowCredentials: true,
		MaxAge:           12 * 3600, // 12 hours
	}))

	// Additional CORS handler for preflight requests
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:5173", 
			"http://localhost:8080",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:3000",
		}
		
		// Check if origin is allowed
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "43200") // 12 hours

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Ping godoc
	// @Summary Health check
	// @Description Check if the server is running
	// @Tags health
	// @Produce json
	// @Success 200 {object} map[string]interface{} "Server is running"
	// @Router /ping [get]
	router.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"ok": true})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// WebSocket endpoint
	router.GET("/ws", gin.WrapH(s.wsServer.Handler()))

	api := router.Group("/api/v1")
	{
		// Authentication routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", s.UserRegister)
			auth.POST("/login", s.UserLogin)
			auth.POST("/refresh", s.RefreshToken)
		}

		authProtected := api.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware(s.cfg.JWT.Secret))
		{
			authProtected.POST("/logout", s.Logout)
		}

		// User routes
		users := api.Group("/users")
		{
			users.GET("/:id", s.GetUserByID)
			users.GET("/", s.GetUserByEmail)
		}

		usersProtected := api.Group("/users")
		usersProtected.Use(middleware.AuthMiddleware(s.cfg.JWT.Secret))
		{
			usersProtected.GET("/profile", s.GetUserProfile)
			usersProtected.PATCH("/profile", s.UpdateUserProfile)
			usersProtected.POST("/profile/avatar", s.UploadAvatar)
		}

		// Conversation routes (protected)
		conversations := api.Group("/conversations")
		conversations.Use(middleware.AuthMiddleware(s.cfg.JWT.Secret))
		{
			conversations.POST("/direct", s.CreateDirectConversation)
			conversations.POST("/group", s.CreateGroupConversation)
			conversations.GET("/", s.GetConversations)
			conversations.GET("/:id", s.GetConversation)
			conversations.POST("/:id/messages", s.SendMessage)
			conversations.GET("/:id/messages", s.GetMessages)
		}

		// Message routes (protected)
		messages := api.Group("/messages")
		messages.Use(middleware.AuthMiddleware(s.cfg.JWT.Secret))
		{
			messages.PATCH("/:id", s.UpdateMessage)
			messages.DELETE("/:id", s.DeleteMessage)
		}

		// Media routes (protected)
		media := api.Group("/media")
		media.Use(middleware.AuthMiddleware(s.cfg.JWT.Secret))
		{
			media.POST("/presign", s.PresignUpload)
			media.POST("/upload", s.UploadFile)
			media.GET("/:id", s.GetMedia)
			media.DELETE("/:id", s.DeleteMedia)
		}

		// Notification routes (protected)
		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware(s.cfg.JWT.Secret))
		{
			notifications.GET("/", s.GetNotifications)
			notifications.GET("/unread", s.GetUnreadNotifications)
			notifications.PATCH("/:id/read", s.MarkNotificationAsRead)
			notifications.PATCH("/read-all", s.MarkAllNotificationsAsRead)
			notifications.GET("/preferences", s.GetNotificationPreferences)
			notifications.PATCH("/preferences", s.UpdateNotificationPreferences)
		}

		// Search routes (protected)
		search := api.Group("/search")
		search.Use(middleware.AuthMiddleware(s.cfg.JWT.Secret))
		{
			search.GET("/", s.Search)
			search.GET("/users", s.SearchUsers)
			search.GET("/messages", s.SearchMessages)
		}
	}

	// Static file serving for uploads (public)
	router.Static("/uploads", "./uploads")
}

// Start server
func (s *ServiceServer) Start() error {
	if err := s.svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// Shutdown server
func (s *ServiceServer) Shutdown(ctx context.Context) error {
	fmt.Print("Shutting down server...")
	if s.wsServer != nil {
		s.wsServer.Close()
	}
	return s.svr.Shutdown(ctx)
}
