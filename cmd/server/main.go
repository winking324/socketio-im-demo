package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"im-demo/internal/config"
	"im-demo/internal/handlers"
	"im-demo/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logger.WithError(err).Warn("Invalid log level, using info")
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.Info("Starting IM server...")

	// Initialize Redis service
	redisService, err := services.NewRedisService(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Redis service")
	}
	defer redisService.Close()

	// Initialize Socket.IO handler
	socketIOHandler, err := handlers.NewSocketIOHandler(cfg, redisService, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Socket.IO handler")
	}

	// Initialize Gin router
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Static file serving for uploads
	router.Static("/uploads", cfg.Upload.UploadDir)

	// Serve web client
	router.Static("/web", "web")
	router.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/web/index.html")
	})

	// Socket.IO endpoint
	router.GET("/socket.io/*any", func(c *gin.Context) {
		socketIOHandler.ServeHTTP(c)
	})
	router.POST("/socket.io/*any", func(c *gin.Context) {
		socketIOHandler.ServeHTTP(c)
	})

	// File upload endpoint
	router.POST("/api/upload", socketIOHandler.HandleFileUpload)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// API endpoints
	api := router.Group("/api")
	{
		// Get room members
		api.GET("/rooms/:roomId/members", func(c *gin.Context) {
			roomID := c.Param("roomId")
			ctx := context.Background()
			members, err := redisService.GetRoomMembers(ctx, roomID)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"members": members})
		})

		// Get message by ID
		api.GET("/messages/:messageId", func(c *gin.Context) {
			messageID := c.Param("messageId")
			ctx := context.Background()
			message, err := redisService.GetMessage(ctx, messageID)
			if err != nil {
				c.JSON(404, gin.H{"error": "Message not found"})
				return
			}
			c.JSON(200, message)
		})
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited")
}
