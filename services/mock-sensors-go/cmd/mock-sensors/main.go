package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"dropsense/mock-sensors/internal/csvsource"
	"dropsense/mock-sensors/internal/sender"
)

const (
	defaultCSVPath      = "/data/sensor-readings.csv"
	defaultTelemetryURL = "http://telemetry-ingestion:8080"

	// 2.5s × 240 ticks (o CSV gerado com DURATION_HOURS=2 e
	// INTERVAL_SECONDS=30 no script do Colab) = 600s = exatamente os
	// 10 minutos do primeiro teste pedido. Esse número é só um
	// default de conveniência, sem nenhuma relação obrigatória com
	// os 30s simulados dentro do CSV - são dois relógios
	// completamente independentes, de propósito (ver internal/sender).
	defaultTickInterval = "2.5s"
	defaultLoop         = true

	// STARTUP_DELAY existe pelo mesmo motivo do Ping em
	// internal/kafka.Producer no telemetry-ingestion: "container
	// saudável" não é garantia de "toda a stack já absorveu a
	// inicialização" - o primeiro tick, que dispara imediatamente por
	// desenho (ver internal/sender.Runner), é exatamente o momento
	// mais provável de bater numa janela fria. Default 0s (sem
	// espera) - configurável por env var pra quem quiser dar mais
	// fôlego antes da primeira rajada.
	defaultStartupDelay = "0s"
)

func main() {
	csvPath := getEnv("CSV_PATH", defaultCSVPath)
	telemetryURL := getEnv("TELEMETRY_INGESTION_URL", defaultTelemetryURL)
	tickInterval := getEnvDuration("TICK_INTERVAL", defaultTickInterval)
	loop := getEnvBool("LOOP", defaultLoop)
	startupDelay := getEnvDuration("STARTUP_DELAY", defaultStartupDelay)

	log.Printf("mock-sensors: carregando %s", csvPath)
	ticks, err := csvsource.LoadTicks(csvPath)
	if err != nil {
		log.Fatalf("mock-sensors: failed to load CSV: %v", err)
	}

	sensorsPerTick := 0
	if len(ticks) > 0 {
		sensorsPerTick = len(ticks[0])
	}
	estimatedSeconds := float64(len(ticks)) * tickInterval.Seconds()
	log.Printf(
		"mock-sensors: %d ticks carregados, ~%d sensores por tick, intervalo real=%s, loop=%v (uma passada completa leva ~%.0fs)",
		len(ticks), sensorsPerTick, tickInterval, loop, estimatedSeconds,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if startupDelay > 0 {
		log.Printf("mock-sensors: aguardando %s antes do primeiro tick (STARTUP_DELAY)...", startupDelay)
		select {
		case <-ctx.Done():
			log.Println("mock-sensors: encerrado durante o delay inicial")
			return
		case <-time.After(startupDelay):
		}
	}

	client := sender.NewClient(telemetryURL)
	runner := sender.NewRunner(client, ticks, tickInterval, loop)

	runner.Run(ctx)
	log.Println("mock-sensors: encerrado")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvDuration(key, fallback string) time.Duration {
	value := getEnv(key, fallback)
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("mock-sensors: invalid %s=%q, using default %s: %v", key, value, fallback, err)
		parsed, _ = time.ParseDuration(fallback)
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("mock-sensors: invalid %s=%q, using default %v: %v", key, value, fallback, err)
		return fallback
	}
	return parsed
}