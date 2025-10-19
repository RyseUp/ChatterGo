package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RyseUp/ChatterGo/config"
	"github.com/RyseUp/ChatterGo/internal/app"
	"github.com/RyseUp/ChatterGo/pkg/database"
	"github.com/gin-gonic/gin"
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

	engine := gin.Default()
	a := app.New(engine, db, cfg)
	a.RegisterRoutes()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: engine,
	}

	// Start Server
	go func() {
		log.Printf("HTTP Listening on %s", srv.Addr)
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Printf("server exited properly")
}
