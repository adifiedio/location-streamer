package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/adifiedio/location-streamer/internal/db"
	"github.com/adifiedio/location-streamer/internal/streamer"
	"github.com/adifiedio/location-streamer/pkg/database"
	"github.com/adifiedio/location-streamer/pkg/queue"
)

func main() {
	// initialize database
	pool, err := database.Connect(context.Background())
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	// initialize kafka consumer
	kafkaBrokers := []string{os.Getenv("KAFKA_BROKER")}
	if kafkaBrokers[0] == "" {
		kafkaBrokers = []string{"localhost:9092"}
	}

	// GroupID "streamer-group" ensures if we run multiple instances of this service they load-balance the events
	consumer := queue.NewConsumer(kafkaBrokers, "location-events", "streamer-group")
	defer consumer.Close()

	// start worker
	worker := streamer.NewWorker(queries, consumer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("shutdown signal received")
		cancel()
	}()

	log.Println("starting streamer service...")
	worker.Start(ctx)
	log.Println("streamer service stopped")
}
