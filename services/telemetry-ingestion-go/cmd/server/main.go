package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"dropsense/telemetry-ingestion/internal/httpapi"
	"dropsense/telemetry-ingestion/internal/ingestion"
)

const (
	pipelineWorkers    = 8   // goroutines consumindo o canal em paralelo
	pipelineBufferSize = 500 // leituras que podem esperar antes de aplicar backpressure
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// StdoutPublisher é o placeholder de hoje. Quando o producer Kafka
	// existir (internal/kafka), essa linha é a ÚNICA coisa que muda -
	// Pipeline e o handler HTTP não sabem nem precisam saber disso.
	publisher := ingestion.StdoutPublisher{}
	pipeline := ingestion.NewPipeline(publisher, pipelineWorkers, pipelineBufferSize)
	pipeline.Start(ctx)

	mux := http.NewServeMux()
	mux.Handle("/readings", httpapi.NewReadingHandler(pipeline))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		log.Println("shutdown signal received, closing server...")
		_ = server.Close()
	}()

	log.Println("telemetry-ingestion listening on :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
