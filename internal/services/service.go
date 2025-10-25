package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/RyseUp/ChatterGo/config"
	"github.com/RyseUp/ChatterGo/internal/middleware"
	"github.com/RyseUp/ChatterGo/internal/repositories"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type ServiceServer struct {
	cfg *config.Config
	r   repositories.Repository
	svr *http.Server
}

func NewServer(cfg *config.Config, r repositories.Repository) (*ServiceServer, error) {
	router := gin.Default()

	s := &ServiceServer{
		cfg: cfg,
		r:   r,
		svr: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: router,
		},
	}

	s.setupRoutes(router)

	return s, nil
}

func (s *ServiceServer) setupRoutes(router *gin.Engine) {
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
		}
	}
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
	return s.svr.Shutdown(ctx)
}
