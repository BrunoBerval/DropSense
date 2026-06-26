package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

type samplePayload struct {
	Foo string `json:"foo"`
}

func TestNewEnvelope_SetsCallerProvidedFields(t *testing.T) {
	env, err := NewEnvelope("SomeEventHappened", 1, "telemetry-ingestion", "corr-123", samplePayload{Foo: "bar"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if env.EventType != "SomeEventHappened" {
		t.Errorf("EventType = %q, want %q", env.EventType, "SomeEventHappened")
	}
	if env.EventVersion != 1 {
		t.Errorf("EventVersion = %d, want 1", env.EventVersion)
	}
	if env.Producer != "telemetry-ingestion" {
		t.Errorf("Producer = %q, want %q", env.Producer, "telemetry-ingestion")
	}
	if env.CorrelationID != "corr-123" {
		t.Errorf("CorrelationID = %q, want %q", env.CorrelationID, "corr-123")
	}
}

func TestNewEnvelope_GeneratesValidEventID(t *testing.T) {
	env, err := NewEnvelope("SomeEventHappened", 1, "telemetry-ingestion", "corr-123", samplePayload{Foo: "bar"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := uuid.Parse(env.EventID); err != nil {
		t.Errorf("EventID %q is not a valid UUID: %v", env.EventID, err)
	}
}

func TestNewEnvelope_TwoCallsProduceDifferentEventIDs(t *testing.T) {
	first, _ := NewEnvelope("SomeEventHappened", 1, "telemetry-ingestion", "corr-123", samplePayload{Foo: "bar"})
	second, _ := NewEnvelope("SomeEventHappened", 1, "telemetry-ingestion", "corr-123", samplePayload{Foo: "bar"})

	if first.EventID == second.EventID {
		t.Errorf("expected distinct EventIDs across calls, got the same value %q twice", first.EventID)
	}
}

func TestNewEnvelope_SetsOccurredAtAsRFC3339NearNow(t *testing.T) {
	env, err := NewEnvelope("SomeEventHappened", 1, "telemetry-ingestion", "corr-123", samplePayload{Foo: "bar"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	occurredAt, err := time.Parse(time.RFC3339, env.OccurredAt)
	if err != nil {
		t.Fatalf("OccurredAt %q is not valid RFC3339: %v", env.OccurredAt, err)
	}

	if d := time.Since(occurredAt); d < -time.Second || d > 2*time.Second {
		t.Errorf("OccurredAt %v looks wrong relative to now (time.Since = %v)", occurredAt, d)
	}
}

func TestNewEnvelope_MarshalsPayloadIntoRawField(t *testing.T) {
	env, err := NewEnvelope("SomeEventHappened", 1, "telemetry-ingestion", "corr-123", samplePayload{Foo: "bar"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var got samplePayload
	if err := json.Unmarshal(env.Payload, &got); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if got.Foo != "bar" {
		t.Errorf("payload.Foo = %q, want %q", got.Foo, "bar")
	}
}

func TestNewEnvelope_WithUnmarshalablePayload_ReturnsError(t *testing.T) {
	_, err := NewEnvelope("SomeEventHappened", 1, "telemetry-ingestion", "corr-123", make(chan int))
	if err == nil {
		t.Fatal("expected an error for an unmarshalable payload, got nil")
	}
}

// Este teste trava o contrato de NOMES DE CAMPO do envelope contra o que
// o README define em "Event Envelope" - se alguém renomear um campo da
// struct Go sem querer (ex.: "EventID" -> json:"event_id"), é este teste
// que avisa antes de qualquer serviço em Python ou C# notar a quebra.
func TestEnvelope_JSON_UsesContractFieldNames(t *testing.T) {
	env, err := NewEnvelope("SomeEventHappened", 1, "telemetry-ingestion", "corr-123", samplePayload{Foo: "bar"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &asMap); err != nil {
		t.Fatalf("failed to unmarshal into map: %v", err)
	}

	for _, field := range []string{"eventId", "eventType", "eventVersion", "occurredAt", "producer", "correlationId", "payload"} {
		if _, ok := asMap[field]; !ok {
			t.Errorf("expected JSON field %q in envelope, got keys %v", field, asMap)
		}
	}
}
