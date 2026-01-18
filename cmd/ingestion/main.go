package main

import (
	"context"
	"log"
	"os"

	"github.com/adifiedio/location-streamer/internal/db"
	"github.com/adifiedio/location-streamer/internal/ingestion"
	"github.com/adifiedio/location-streamer/pkg/auth"
	"github.com/adifiedio/location-streamer/pkg/database"
	"github.com/adifiedio/location-streamer/pkg/queue"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Initialize Database
	pool, err := database.Connect(context.Background())
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	// 2. Initialize Kafka Producer
	kafkaBrokers := []string{os.Getenv("KAFKA_BROKER")}
	if kafkaBrokers[0] == "" {
		kafkaBrokers = []string{"localhost:9092"}
	}
	producer := queue.NewProducer(kafkaBrokers, "location-events")
	defer producer.Close()

	// 3. Initialize Handler
	handler := ingestion.NewHandler(queries, producer)

	// 4. Setup Router
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "ingestion-service"})
	})

	api := r.Group("/api/v1")
	api.Use(auth.Middleware())
	{
		api.POST("/locations", handler.IngestLocation)
	}

	// 5. Start Server
	// We run on a different port than tenant service
	port := os.Getenv("INGESTION_PORT")
	if port == "" {
		port = "8082"
	}
	log.Printf("Starting Ingestion Service on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
