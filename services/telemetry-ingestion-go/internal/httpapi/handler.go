package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"dropsense/telemetry-ingestion/internal/ingestion"
)

// readingPayload mirrors the exact JSON shape sensors send (see
// docs/01-CONTEXTO-PROJETO.md). Kept separate from ingestion.SoilReading
// on purpose: this struct is about the wire format (JSON field names,
// timestamp as RFC3339 string); the domain struct is about the
// Go-native shape (time.Time). Decoding is the translation between the
// two - this handler IS the Anti-Corruption Layer at the network edge.
type readingPayload struct {
	SensorID               string  `json:"sensorId"`
	ZoneID                 string  `json:"zoneId"`
	SoilMoisturePercent    float64 `json:"soilMoisturePercent"`
	SoilTemperatureCelsius float64 `json:"soilTemperatureCelsius"`
	MeasuredAt             string  `json:"measuredAt"`
}

// Submitter is the only thing this handler needs from a pipeline,
// narrowed to a single method on purpose: handler tests use a fake
// Submitter instead of a real ingestion.Pipeline, so they run with zero
// goroutines and zero channels involved.
type Submitter interface {
	Submit(reading ingestion.SoilReading)
}

type ReadingHandler struct {
	pipeline Submitter
}

func NewReadingHandler(pipeline Submitter) *ReadingHandler {
	return &ReadingHandler{pipeline: pipeline}
}

func (h *ReadingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload readingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	measuredAt, err := time.Parse(time.RFC3339, payload.MeasuredAt)
	if err != nil {
		http.Error(w, "measuredAt must be a valid RFC3339 timestamp", http.StatusBadRequest)
		return
	}

	reading := ingestion.SoilReading{
		SensorID:               payload.SensorID,
		ZoneID:                 payload.ZoneID,
		SoilMoisturePercent:    payload.SoilMoisturePercent,
		SoilTemperatureCelsius: payload.SoilTemperatureCelsius,
		MeasuredAt:             measuredAt,
	}

	// Validação síncrona, de propósito: é checagem de CPU pura (sem
	// I/O), então não compete com o motivo de existir do canal (que é
	// proteger contra a parte LENTA - o publish no Kafka). "Falhar
	// rápido na porta" também significa devolver 400 pro
	// sensor/gateway na hora, em vez de aceitar (202) e descartar
	// silenciosamente depois.
	if err := reading.Validate(); err != nil {
		log.Printf("[discarded] sensor=%s zone=%s reason=%v", reading.SensorID, reading.ZoneID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.pipeline.Submit(reading)
	w.WriteHeader(http.StatusAccepted)
}
