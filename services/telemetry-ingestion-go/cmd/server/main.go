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
	"time"

	"dropsense/telemetry-ingestion/internal/httpapi"
	"dropsense/telemetry-ingestion/internal/ingestion"
	"dropsense/telemetry-ingestion/internal/kafka"
	"dropsense/telemetry-ingestion/internal/weather"
)

const (
	pipelineWorkers    = 8   // goroutines consumindo o canal em paralelo
	pipelineBufferSize = 500 // leituras que podem esperar antes de aplicar backpressure

	// Defaults pensados pra rodar dentro do docker-compose: "kafka" é
	// o nome do serviço na mesma rede. KAFKA_BROKERS aceita uma lista
	// separada por vírgula (ex.: "broker1:9092,broker2:9092") para
	// quando este projeto crescer para múltiplos brokers. Não existe
	// mais KAFKA_TOPIC: agora que o Producer publica em mais de um
	// tópico, cada tópico é constante junto da definição do seu
	// evento (ver internal/events), não um detalhe de deploy.
	defaultKafkaBrokers = "kafka:9092"

	// Forecast não muda a cada segundo; 30min equilibra "dado
	// razoavelmente fresco" com não estourar o uso justo da API
	// gratuita do Open-Meteo.
	defaultWeatherPollInterval = 30 * time.Minute
)

func main() {
	log.Printf("GOMAXPROCS=%d NumCPU=%d", runtime.GOMAXPROCS(0), runtime.NumCPU())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	brokers := strings.Split(getEnv("KAFKA_BROKERS", defaultKafkaBrokers), ",")

	// StdoutPublisher era o placeholder de antes. O producer Kafka
	// real (internal/kafka) implementa tanto ingestion.Publisher
	// quanto weather.ForecastPublisher - a mesma conexão atende as
	// duas pipelines abaixo.
	publisher, err := kafka.NewProducer(brokers)
	if err != nil {
		log.Fatalf("failed to connect to kafka: %v", err)
	}
	defer publisher.Close()

	pipeline := ingestion.NewPipeline(publisher, pipelineWorkers, pipelineBufferSize)
	pipeline.Start(ctx)

	// Consulta periódica de previsão do tempo, publicando
	// WeatherForecastUpdated - "paralelamente, em intervalos
	// regulares, esse mesmo serviço consulta uma API externa de
	// previsão do tempo", como o README descreve na Parte 2.
	weatherScheduler := weather.NewScheduler(
		weather.NewOpenMeteoClient(),
		publisher,
		weather.Zones(),
		getEnvDuration("WEATHER_POLL_INTERVAL", defaultWeatherPollInterval),
	)
	weatherScheduler.Start(ctx)

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

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid %s=%q, using default %s: %v", key, value, fallback, err)
		return fallback
	}
	return parsed
}
