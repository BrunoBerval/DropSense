package sender

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"dropsense/mock-sensors/internal/csvsource"
)

// fakeDoer grava a última requisição que recebeu e devolve uma
// resposta canned - mesmo papel de fakeProduceClient (Go),
// IRawProducer (C#) e o client injetável do Python: testar a
// montagem da requisição sem precisar de um servidor real.
type fakeDoer struct {
	lastRequest *http.Request
	lastBody    []byte
	statusCode  int
	err         error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastRequest = req
	if req.Body != nil {
		f.lastBody, _ = io.ReadAll(req.Body)
	}
	status := f.statusCode
	if status == 0 {
		status = http.StatusAccepted
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func validReading() csvsource.Reading {
	return csvsource.Reading{
		SensorID:               "sensor-001-001",
		ZoneID:                 "zone-001",
		SoilMoisturePercent:    40.5,
		SoilTemperatureCelsius: 20.3,
	}
}

func TestSend_PostsToReadingsEndpoint(t *testing.T) {
	doer := &fakeDoer{}
	client := &Client{httpClient: doer, baseURL: "http://telemetry-ingestion:8080"}

	if err := client.Send(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := doer.lastRequest.URL.String(); got != "http://telemetry-ingestion:8080/readings" {
		t.Errorf("URL = %q, want %q", got, "http://telemetry-ingestion:8080/readings")
	}
	if doer.lastRequest.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", doer.lastRequest.Method)
	}
}

func TestSend_SetsJSONContentType(t *testing.T) {
	doer := &fakeDoer{}
	client := &Client{httpClient: doer, baseURL: "http://telemetry-ingestion:8080"}

	if err := client.Send(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := doer.lastRequest.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
}

func TestSend_BodyMatchesReadingFields(t *testing.T) {
	doer := &fakeDoer{}
	client := &Client{httpClient: doer, baseURL: "http://telemetry-ingestion:8080"}

	if err := client.Send(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var got wirePayload
	if err := json.Unmarshal(doer.lastBody, &got); err != nil {
		t.Fatalf("failed to unmarshal sent body: %v", err)
	}

	if got.SensorID != "sensor-001-001" {
		t.Errorf("SensorID = %q, want %q", got.SensorID, "sensor-001-001")
	}
	if got.ZoneID != "zone-001" {
		t.Errorf("ZoneID = %q, want %q", got.ZoneID, "zone-001")
	}
	if got.SoilMoisturePercent != 40.5 {
		t.Errorf("SoilMoisturePercent = %v, want 40.5", got.SoilMoisturePercent)
	}
	if got.SoilTemperatureCelsius != 20.3 {
		t.Errorf("SoilTemperatureCelsius = %v, want 20.3", got.SoilTemperatureCelsius)
	}
}

func TestSend_SetsMeasuredAtToSendTime(t *testing.T) {
	doer := &fakeDoer{}
	client := &Client{httpClient: doer, baseURL: "http://telemetry-ingestion:8080"}

	before := time.Now().UTC()
	if err := client.Send(context.Background(), validReading()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	after := time.Now().UTC()

	var got wirePayload
	if err := json.Unmarshal(doer.lastBody, &got); err != nil {
		t.Fatalf("failed to unmarshal sent body: %v", err)
	}

	measuredAt, err := time.Parse(time.RFC3339, got.MeasuredAt)
	if err != nil {
		t.Fatalf("MeasuredAt %q is not valid RFC3339: %v", got.MeasuredAt, err)
	}
	if measuredAt.Before(before.Add(-time.Second)) || measuredAt.After(after.Add(time.Second)) {
		t.Errorf("MeasuredAt %v not close to send time (between %v and %v)", measuredAt, before, after)
	}
}

func TestSend_WhenServerReturnsNon202_ReturnsError(t *testing.T) {
	doer := &fakeDoer{statusCode: http.StatusBadRequest}
	client := &Client{httpClient: doer, baseURL: "http://telemetry-ingestion:8080"}

	if err := client.Send(context.Background(), validReading()); err == nil {
		t.Fatal("expected an error for a non-202 response, got nil")
	}
}

func TestSend_WhenRequestFails_ReturnsError(t *testing.T) {
	doer := &fakeDoer{err: errors.New("connection refused")}
	client := &Client{httpClient: doer, baseURL: "http://telemetry-ingestion:8080"}

	if err := client.Send(context.Background(), validReading()); err == nil {
		t.Fatal("expected an error, got nil")
	}
}
