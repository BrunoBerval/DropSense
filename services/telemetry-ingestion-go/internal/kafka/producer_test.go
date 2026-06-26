package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"dropsense/telemetry-ingestion/internal/events"
	"dropsense/telemetry-ingestion/internal/ingestion"
)

// fakeProduceClient grava todo record que recebe, e deixa o teste decidir
// se a "publicação" deve falhar ou não - mesmo padrão de fakeSubmitter
// (handler_test.go) e recordingPublisher (pipeline_test.go) já usados no
// resto do projeto: testar a lógica de montagem da mensagem sem depender
// de um broker Kafka real.
type fakeProduceClient struct {
	records []*kgo.Record
	err     error
}

func (f *fakeProduceClient) ProduceSync(_ context.Context, rs ...*kgo.Record) kgo.ProduceResults {
	f.records = append(f.records, rs...)
	results := make(kgo.ProduceResults, len(rs))
	for i, r := range rs {
		results[i] = kgo.ProduceResult{Record: r, Err: f.err}
	}
	return results
}

func (f *fakeProduceClient) Close() {}

func validReading() ingestion.SoilReading {
	return ingestion.SoilReading{
		SensorID:               "sensor-04812",
		ZoneID:                 "zone-042",
		SoilMoisturePercent:    38.5,
		SoilTemperatureCelsius: 24.1,
		MeasuredAt:             time.Date(2026, 6, 23, 14, 30, 0, 0, time.UTC),
	}
}

func TestPublish_SendsToConfiguredTopic(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client, topic: "telemetry.readings.v1"}

	if err := producer.Publish(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(client.records) != 1 {
		t.Fatalf("expected 1 record produced, got %d", len(client.records))
	}
	if got := client.records[0].Topic; got != "telemetry.readings.v1" {
		t.Errorf("Topic = %q, want %q", got, "telemetry.readings.v1")
	}
}

func TestPublish_UsesSensorIDAsPartitionKey(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client, topic: "telemetry.readings.v1"}

	if err := producer.Publish(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := string(client.records[0].Key); got != "sensor-04812" {
		t.Errorf("Key = %q, want %q", got, "sensor-04812")
	}
}

func TestPublish_WrapsPayloadInEnvelopeWithReadmeContractFields(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client, topic: "telemetry.readings.v1"}

	if err := producer.Publish(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var envelope events.Envelope
	if err := json.Unmarshal(client.records[0].Value, &envelope); err != nil {
		t.Fatalf("produced value is not a valid envelope: %v", err)
	}

	if envelope.EventType != events.SoilReadingEventType {
		t.Errorf("EventType = %q, want %q", envelope.EventType, events.SoilReadingEventType)
	}
	if envelope.EventVersion != events.SoilReadingEventVersion {
		t.Errorf("EventVersion = %d, want %d", envelope.EventVersion, events.SoilReadingEventVersion)
	}
	if envelope.Producer != "telemetry-ingestion" {
		t.Errorf("Producer = %q, want %q", envelope.Producer, "telemetry-ingestion")
	}
	if envelope.CorrelationID == "" {
		t.Error("expected a non-empty CorrelationID")
	}

	var payload events.SoilReadingPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("envelope payload is not a valid SoilReadingPayload: %v", err)
	}
	if payload.SensorID != "sensor-04812" {
		t.Errorf("payload.SensorID = %q, want %q", payload.SensorID, "sensor-04812")
	}
	if payload.MeasuredAt != "2026-06-23T14:30:00Z" {
		t.Errorf("payload.MeasuredAt = %q, want %q", payload.MeasuredAt, "2026-06-23T14:30:00Z")
	}
}

func TestPublish_WhenBrokerReturnsError_PropagatesIt(t *testing.T) {
	client := &fakeProduceClient{err: errors.New("broker unavailable")}
	producer := &Producer{client: client, topic: "telemetry.readings.v1"}

	err := producer.Publish(context.Background(), validReading())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
