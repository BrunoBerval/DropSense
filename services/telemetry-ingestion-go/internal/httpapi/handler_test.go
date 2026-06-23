package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"dropsense/telemetry-ingestion/internal/ingestion"
)

// fakeSubmitter records every reading submitted to it. httptest invokes
// handlers synchronously within a single goroutine per test, but the
// mutex costs nothing and keeps this safe if that ever changes.
type fakeSubmitter struct {
	mu        sync.Mutex
	submitted []ingestion.SoilReading
}

func (f *fakeSubmitter) Submit(reading ingestion.SoilReading) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.submitted = append(f.submitted, reading)
}

func (f *fakeSubmitter) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.submitted)
}

func postReading(handler http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/readings", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestServeHTTP_WithValidPayload_Returns202AndSubmitsToPipeline(t *testing.T) {
	submitter := &fakeSubmitter{}
	handler := NewReadingHandler(submitter)

	body := `{
		"sensorId": "sensor-04812",
		"zoneId": "zone-042",
		"soilMoisturePercent": 38.5,
		"soilTemperatureCelsius": 24.1,
		"measuredAt": "2026-06-23T14:30:00Z"
	}`

	rec := postReading(handler, body)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if submitter.count() != 1 {
		t.Fatalf("expected 1 reading submitted to the pipeline, got %d", submitter.count())
	}
}

func TestServeHTTP_WithPhysicallyImpossibleMoisture_Returns400AndDoesNotSubmit(t *testing.T) {
	submitter := &fakeSubmitter{}
	handler := NewReadingHandler(submitter)

	body := `{
		"sensorId": "sensor-04812",
		"zoneId": "zone-042",
		"soilMoisturePercent": 150,
		"soilTemperatureCelsius": 24.1,
		"measuredAt": "2026-06-23T14:30:00Z"
	}`

	rec := postReading(handler, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if submitter.count() != 0 {
		t.Fatalf("expected reading to be rejected BEFORE reaching the pipeline, got %d submitted", submitter.count())
	}
}

func TestServeHTTP_WithMalformedJSON_Returns400(t *testing.T) {
	submitter := &fakeSubmitter{}
	handler := NewReadingHandler(submitter)

	rec := postReading(handler, `{this is not json`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestServeHTTP_WithInvalidTimestamp_Returns400(t *testing.T) {
	submitter := &fakeSubmitter{}
	handler := NewReadingHandler(submitter)

	body := `{
		"sensorId": "sensor-04812",
		"zoneId": "zone-042",
		"soilMoisturePercent": 38.5,
		"soilTemperatureCelsius": 24.1,
		"measuredAt": "not-a-valid-timestamp"
	}`

	rec := postReading(handler, body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestServeHTTP_WithWrongHTTPMethod_Returns405(t *testing.T) {
	submitter := &fakeSubmitter{}
	handler := NewReadingHandler(submitter)

	req := httptest.NewRequest(http.MethodGet, "/readings", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
