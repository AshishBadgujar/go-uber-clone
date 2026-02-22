package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081")
	rabbitmqUrl = env.GetString("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
)

func main() {
	log.Println("Starting API Gateway")

	mux := http.NewServeMux()

	rabbitmq, err := messaging.NewRabbitMQ(rabbitmqUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("Connected to RabbitMQ")

	mux.HandleFunc("POST /trip/preview", enableCors(handleTripPreview))
	mux.HandleFunc("POST /trip/start", enableCors(handleTripStart))
	mux.HandleFunc("/ws/drivers", func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq)
	})
	mux.HandleFunc("/ws/riders", func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq)
	})
	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Println("Starting server on port", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Failed to start server: %v", err)
	case sig := <-shutdown:
		log.Printf("Server is shutting down due to %v", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Failed to shutdown server: %v", err)
			server.Close()
		}
		log.Println("Server shutdown gracefully")
	}
}
