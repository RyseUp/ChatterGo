package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RyseUp/ChatterGo/config"
	"github.com/RyseUp/ChatterGo/internal/models"
	"github.com/RyseUp/ChatterGo/internal/repositories/postgres"
	"github.com/RyseUp/ChatterGo/internal/services"
	"github.com/RyseUp/ChatterGo/pkg/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Open(cfg.Database.GetDatabaseDSN())
	if err != nil {
		log.Fatalf("failed to connect DB: %v", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Create repository
	repo := postgres.NewQueries(db, cfg)

	// Create and configure server
	server, err := services.NewServer(cfg, repo)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Start Server
	go func() {
		log.Printf("HTTP Listening on :%d", cfg.Server.Port)
		if err = server.Start(); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Printf("server exited properly")
}
