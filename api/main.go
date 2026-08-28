package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"streampulse/api/handlers"
	"streampulse/api/kafka"
)

func main() {
	port := getEnv("PORT", "4002")
	brokers := getEnv("KAFKA_BROKERS", "localhost:9093")
	topic := getEnv("KAFKA_TOPIC", "events")

	producer := kafka.NewProducer(brokers, topic)
	defer producer.Close()

	eventHandler := handlers.NewEventHandler(producer)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", eventHandler.Ingest)
	mux.HandleFunc("GET /healthz", handlers.Health)
	mux.HandleFunc("GET /readyz", eventHandler.Ready)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("streampulse api listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
