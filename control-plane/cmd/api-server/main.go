// control-plane/cmd/api-server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gon-cloud-platform/control-plane/internal/api/routes"
	"gon-cloud-platform/control-plane/internal/database"
	"gon-cloud-platform/control-plane/internal/messaging"
	"gon-cloud-platform/control-plane/internal/utils"

	"github.com/gin-gonic/gin"
)

type Server struct {
	DB     *database.Connection
	Router *gin.Engine
	Config *utils.Config
	MQ     *messaging.RabbitMQ
}

func main() {
	// Load configuration
	config, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := utils.NewLogger(config.LogLevel)

	// Initialize database connection
	db, err := database.NewConnection(config.Database)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		logger.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize message queue
	mq, err := messaging.NewRabbitMQ(config.RabbitMQ)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer mq.Close()

	// Create server instance
	server := &Server{
		DB:     db,
		Config: config,
		MQ:     mq,
	}

	// Setup Gin router
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	server.Router = gin.New()
	server.Router.Use(gin.Logger())
	server.Router.Use(gin.Recovery())

	// Setup routes
	routes.SetupRoutes(server.Router, server.DB, server.MQ, config)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         ":" + config.Server.Port,
		Handler:      server.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting API server on port %s", config.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
