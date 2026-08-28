package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"streampulse/processor/aggregator"
	ch "streampulse/processor/clickhouse"
	"streampulse/processor/consumer"
	rds "streampulse/processor/redis"
	"streampulse/processor/websocket"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisClient := rds.NewClient(getEnv("REDIS_ADDR", "localhost:6400"))
	chClient, err := ch.NewClient(
		getEnv("CLICKHOUSE_ADDR", "localhost:9004"),
		getEnv("CLICKHOUSE_DB", "streampulse"),
	)
	if err != nil {
		log.Fatalf("clickhouse connection failed: %v", err)
	}

	hub := websocket.NewHub()
	go hub.Run()

	agg := aggregator.New(redisClient, chClient, hub)

	c := consumer.New(
		getEnv("KAFKA_BROKERS", "localhost:9093"),
		getEnv("KAFKA_TOPIC", "events"),
		getEnv("KAFKA_GROUP_ID", "streampulse-processor"),
		agg.HandleEvent,
	)
	go c.Run(ctx)

	// Broadcast aggregated stats to all connected dashboards every second.
	go agg.RunBroadcastLoop(ctx, 1*time.Second)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	wsPort := getEnv("WS_PORT", "4003")
	srv := &http.Server{Addr: ":" + wsPort, Handler: mux}

	go func() {
		log.Printf("streampulse processor websocket server listening on :%s", wsPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ws server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down processor...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
