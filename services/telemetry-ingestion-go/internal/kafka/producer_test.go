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
	"dropsense/telemetry-ingestion/internal/weather"
)

// fakeProduceClient grava todo record que recebe, e deixa o teste
// decidir se a "publicação" deve falhar ou não - mesmo padrão de
// fakeSubmitter (handler_test.go) e recordingPublisher
// (pipeline_test.go) já usados no resto do projeto.
type fakeProduceClient struct {
	records []*kgo.Record
	err     error
	pingErr error
}

func (f *fakeProduceClient) ProduceSync(_ context.Context, rs ...*kgo.Record) kgo.ProduceResults {
	f.records = append(f.records, rs...)
	results := make(kgo.ProduceResults, len(rs))
	for i, r := range rs {
		results[i] = kgo.ProduceResult{Record: r, Err: f.err}
	}
	return results
}

func (f *fakeProduceClient) Ping(_ context.Context) error {
	return f.pingErr
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

func validForecast() weather.Forecast {
	return weather.Forecast{
		WindowHours:                12,
		RainProbabilityPercent:     80,
		ForecastTemperatureCelsius: 29.5,
	}
}

// --- Publish (SoilReadingRegistered) ---

func TestPublish_SendsToConfiguredTopic(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client}

	if err := producer.Publish(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(client.records) != 1 {
		t.Fatalf("expected 1 record produced, got %d", len(client.records))
	}
	if got := client.records[0].Topic; got != events.SoilReadingTopic {
		t.Errorf("Topic = %q, want %q", got, events.SoilReadingTopic)
	}
}

func TestPublish_UsesSensorIDAsPartitionKey(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client}

	if err := producer.Publish(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := string(client.records[0].Key); got != "sensor-04812" {
		t.Errorf("Key = %q, want %q", got, "sensor-04812")
	}
}

func TestPublish_WrapsPayloadInEnvelopeWithReadmeContractFields(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client}

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
	producer := &Producer{client: client}

	err := producer.Publish(context.Background(), validReading())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// --- PublishWeatherForecast (WeatherForecastUpdated) ---

func TestPublishWeatherForecast_SendsToWeatherTopic(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client}

	if err := producer.PublishWeatherForecast(context.Background(), "zone-042", validForecast()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(client.records) != 1 {
		t.Fatalf("expected 1 record produced, got %d", len(client.records))
	}
	if got := client.records[0].Topic; got != events.WeatherForecastTopic {
		t.Errorf("Topic = %q, want %q", got, events.WeatherForecastTopic)
	}
}

func TestPublishWeatherForecast_UsesZoneIDAsPartitionKey(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client}

	if err := producer.PublishWeatherForecast(context.Background(), "zone-042", validForecast()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := string(client.records[0].Key); got != "zone-042" {
		t.Errorf("Key = %q, want %q", got, "zone-042")
	}
}

func TestPublishWeatherForecast_WrapsPayloadInEnvelopeWithReadmeContractFields(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client}

	if err := producer.PublishWeatherForecast(context.Background(), "zone-042", validForecast()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var envelope events.Envelope
	if err := json.Unmarshal(client.records[0].Value, &envelope); err != nil {
		t.Fatalf("produced value is not a valid envelope: %v", err)
	}

	if envelope.EventType != events.WeatherForecastEventType {
		t.Errorf("EventType = %q, want %q", envelope.EventType, events.WeatherForecastEventType)
	}
	if envelope.Producer != "telemetry-ingestion" {
		t.Errorf("Producer = %q, want %q", envelope.Producer, "telemetry-ingestion")
	}

	var payload events.WeatherForecastPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("envelope payload is not a valid WeatherForecastPayload: %v", err)
	}
	if payload.ZoneID != "zone-042" {
		t.Errorf("payload.ZoneID = %q, want %q", payload.ZoneID, "zone-042")
	}
	if payload.Source != "open-meteo" {
		t.Errorf("payload.Source = %q, want %q", payload.Source, "open-meteo")
	}
}

func TestPublishWeatherForecast_WhenBrokerReturnsError_PropagatesIt(t *testing.T) {
	client := &fakeProduceClient{err: errors.New("broker unavailable")}
	producer := &Producer{client: client}

	err := producer.PublishWeatherForecast(context.Background(), "zone-042", validForecast())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// --- Ping ---

func TestPing_Succeeds_WhenClientPingSucceeds(t *testing.T) {
	client := &fakeProduceClient{}
	producer := &Producer{client: client}

	if err := producer.Ping(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPing_PropagatesClientError(t *testing.T) {
	client := &fakeProduceClient{pingErr: errors.New("broker unreachable")}
	producer := &Producer{client: client}

	if err := producer.Ping(context.Background()); err == nil {
		t.Fatal("expected an error, got nil")
	}
}