package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"dropsense/telemetry-ingestion/internal/httpapi"
	"dropsense/telemetry-ingestion/internal/ingestion"
	"dropsense/telemetry-ingestion/internal/kafka"
)

const (
	pipelineWorkers    = 8   // goroutines consumindo o canal em paralelo
	pipelineBufferSize = 500 // leituras que podem esperar antes de aplicar backpressure

	// Defaults pensados pra rodar dentro do docker-compose: "kafka" é o
	// nome do serviço na mesma rede, e o tópico é o definido na tabela
	// de tópicos do README. KAFKA_BROKERS aceita uma lista separada por
	// vírgula (ex.: "broker1:9092,broker2:9092") para quando este
	// projeto crescer para múltiplos brokers.
	defaultKafkaBrokers = "kafka:9092"
	defaultKafkaTopic   = "telemetry.readings.v1"
)

func main() {
	log.Printf("GOMAXPROCS=%d NumCPU=%d", runtime.GOMAXPROCS(0), runtime.NumCPU())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	brokers := strings.Split(getEnv("KAFKA_BROKERS", defaultKafkaBrokers), ",")
	topic := getEnv("KAFKA_TOPIC", defaultKafkaTopic)

	// StdoutPublisher era o placeholder de antes. Agora que o producer
	// Kafka real existe (internal/kafka), essa é a ÚNICA linha que
	// muda - Pipeline e o handler HTTP não sabem nem precisam saber
	// disso, exatamente como o comentário original já previa.
	publisher, err := kafka.NewProducer(brokers, topic)
	if err != nil {
		log.Fatalf("failed to connect to kafka: %v", err)
	}
	defer publisher.Close()

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

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
