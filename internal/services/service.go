package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/RyseUp/ChatterGo/config"
	"github.com/RyseUp/ChatterGo/internal/repositories"
	"github.com/gin-gonic/gin"
)

type ServiceServer struct {
	cfg *config.Config
	r   repositories.Repository
	svr *http.Server
}

func NewServer(cfg *config.Config, r repositories.Repository) (*ServiceServer, error) {
	router := gin.Default()

	// Create service server instance
	s := &ServiceServer{
		cfg: cfg,
		r:   r,
		svr: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: router,
		},
	}

	// Set up routes
	s.setupRoutes(router)

	return s, nil
}

// setupRoutes configures all the routes for the application
func (s *ServiceServer) setupRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"ok": true})
	})

	// API v1 routes
	api := router.Group("/api/v1")
	{
		// User routes
		users := api.Group("/users")
		{
			users.POST("/register", s.UserRegister) // POST /api/v1/users/register
			users.GET("/:id", s.GetUserByID)        // GET /api/v1/users/:id
			users.GET("/", s.GetUserByEmail)        // GET /api/v1/users?email=...
		}
	}
}

// Start starts the HTTP server
func (s *ServiceServer) Start() error {
	if err := s.svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *ServiceServer) Shutdown(ctx context.Context) error {
	fmt.Print("Shutting down server...")
	return s.svr.Shutdown(ctx)
}
