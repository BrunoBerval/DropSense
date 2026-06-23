package ingestion

import (
	"context"
	"log"
	"time"
)

// Publisher abstracts what happens to a validated reading after it
// leaves the channel pipeline. The real implementation - publishing a
// SoilReadingRegistered event to the telemetry.readings.v1 Kafka topic
// - doesn't exist yet; it will live in a future internal/kafka package
// that satisfies this same interface. Keeping it as an interface here
// is what lets Pipeline be tested today with zero Kafka dependency.
type Publisher interface {
	Publish(ctx context.Context, reading SoilReading) error
}

// StdoutPublisher is a placeholder implementation used until the Kafka
// producer exists. It makes the service runnable end-to-end today
// (HTTP -> validate -> channel -> "publish") without any broker.
type StdoutPublisher struct{}

func (StdoutPublisher) Publish(_ context.Context, reading SoilReading) error {
	log.Printf(
		"[stub publish] sensor=%s zone=%s moisture=%.1f%% temp=%.1fC measuredAt=%s",
		reading.SensorID,
		reading.ZoneID,
		reading.SoilMoisturePercent,
		reading.SoilTemperatureCelsius,
		reading.MeasuredAt.Format(time.RFC3339),
	)
	return nil
}
