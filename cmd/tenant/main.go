package main

import (
	"context"
	"log"
	"os"

	"github.com/adifiedio/location-streamer/internal/db"
	"github.com/adifiedio/location-streamer/internal/tenant"
	"github.com/adifiedio/location-streamer/pkg/auth"
	"github.com/adifiedio/location-streamer/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. initialize database
	pool, err := database.Connect(context.Background())
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	// 2. initialize service layer
	queries := db.New(pool)
	handler := tenant.NewHandler(queries, pool)

	// 3. setup router
	r := gin.Default()

	// health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "tenant-service"})
	})

	// protected routes
	api := r.Group("/api/v1")
	api.Use(auth.Middleware()) // Enforce Auth
	{
		// registration (anyone with a token can create a tenant)
		api.POST("/register", handler.Register)

		// user management (tenant admins only, logic inside handler)
		api.POST("/users", handler.AddUser)

		// system admin (optional)
		api.GET("/tenants", handler.ListTenants)
	}

	// 4. start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Starting Tenant Service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
