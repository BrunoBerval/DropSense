package events

import (
	"encoding/json"
	"testing"
	"time"

	"dropsense/telemetry-ingestion/internal/ingestion"
)

func readmeExampleReading() ingestion.SoilReading {
	// Os valores abaixo são exatamente os do exemplo de payload do
	// SoilReadingRegistered no README, seção "EVENT CONTRACTS - SOURCE
	// OF TRUTH". Usar os mesmos valores aqui é o que torna o teste
	// abaixo uma conferência de verdade contra o contrato documentado,
	// não só contra "alguma" struct.
	return ingestion.SoilReading{
		SensorID:               "sensor-04812",
		ZoneID:                 "zone-042",
		SoilMoisturePercent:    38.5,
		SoilTemperatureCelsius: 24.1,
		MeasuredAt:             time.Date(2026, 6, 23, 14, 30, 0, 0, time.UTC),
	}
}

func TestNewSoilReadingPayload_TranslatesAllFields(t *testing.T) {
	payload := NewSoilReadingPayload(readmeExampleReading())

	if payload.SensorID != "sensor-04812" {
		t.Errorf("SensorID = %q, want %q", payload.SensorID, "sensor-04812")
	}
	if payload.ZoneID != "zone-042" {
		t.Errorf("ZoneID = %q, want %q", payload.ZoneID, "zone-042")
	}
	if payload.SoilMoisturePercent != 38.5 {
		t.Errorf("SoilMoisturePercent = %v, want 38.5", payload.SoilMoisturePercent)
	}
	if payload.SoilTemperatureCelsius != 24.1 {
		t.Errorf("SoilTemperatureCelsius = %v, want 24.1", payload.SoilTemperatureCelsius)
	}
	if payload.MeasuredAt != "2026-06-23T14:30:00Z" {
		t.Errorf("MeasuredAt = %q, want %q", payload.MeasuredAt, "2026-06-23T14:30:00Z")
	}
}

// Este é O teste que responde à pergunta "o formato bate com o README?"
// de forma literal: serializa o payload e compara, byte a byte, com o
// JSON exemplo documentado em "EVENT CONTRACTS - SOURCE OF TRUTH" >
// SoilReadingRegistered > Payload. Se o contrato publicado divergir do
// README, é aqui que quebra primeiro - antes de qualquer serviço em
// Python ou C# notar em produção.
func TestSoilReadingPayload_JSON_MatchesReadmeContract(t *testing.T) {
	got, err := json.Marshal(NewSoilReadingPayload(readmeExampleReading()))
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	want := `{"sensorId":"sensor-04812","zoneId":"zone-042","soilMoisturePercent":38.5,"soilTemperatureCelsius":24.1,"measuredAt":"2026-06-23T14:30:00Z"}`

	if string(got) != want {
		t.Errorf("JSON payload mismatch.\ngot:  %s\nwant: %s", got, want)
	}
}

func TestNewSoilReadingPayload_FormatsMeasuredAtAsUTC(t *testing.T) {
	// measuredAt chega no domínio em UTC (handler.go já garante isso ao
	// usar time.Parse com RFC3339), mas se algum dia chegar em outro
	// fuso, o payload de fio deve normalizar para UTC mesmo assim - é
	// o "Z" no final que o README usa, não um offset como "-03:00".
	loc := time.FixedZone("BRT", -3*60*60)
	reading := readmeExampleReading()
	reading.MeasuredAt = time.Date(2026, 6, 23, 11, 30, 0, 0, loc) // mesmo instante, em -03:00

	payload := NewSoilReadingPayload(reading)

	if payload.MeasuredAt != "2026-06-23T14:30:00Z" {
		t.Errorf("MeasuredAt = %q, want %q (normalizado para UTC)", payload.MeasuredAt, "2026-06-23T14:30:00Z")
	}
}
