package events

import (
	"encoding/json"
	"testing"

	"dropsense/telemetry-ingestion/internal/weather"
)

func readmeExampleForecast() weather.Forecast {
	// Valores exatamente do exemplo de payload do WeatherForecastUpdated
	// no README, seção "EVENT CONTRACTS - SOURCE OF TRUTH".
	return weather.Forecast{
		WindowHours:                12,
		RainProbabilityPercent:     80,
		ForecastTemperatureCelsius: 29.5,
	}
}

func TestNewWeatherForecastPayload_TranslatesAllFields(t *testing.T) {
	payload := NewWeatherForecastPayload("zone-042", readmeExampleForecast())

	if payload.ZoneID != "zone-042" {
		t.Errorf("ZoneID = %q, want %q", payload.ZoneID, "zone-042")
	}
	if payload.RainProbabilityPercent != 80 {
		t.Errorf("RainProbabilityPercent = %d, want 80", payload.RainProbabilityPercent)
	}
	if payload.ForecastTemperatureCelsius != 29.5 {
		t.Errorf("ForecastTemperatureCelsius = %v, want 29.5", payload.ForecastTemperatureCelsius)
	}
	if payload.ForecastWindowHours != 12 {
		t.Errorf("ForecastWindowHours = %d, want 12", payload.ForecastWindowHours)
	}
	if payload.Source != "open-meteo" {
		t.Errorf("Source = %q, want %q", payload.Source, "open-meteo")
	}
}

// Mesmo papel de TestSoilReadingPayload_JSON_MatchesReadmeContract:
// trava o contrato publicado contra o exemplo literal do README. Se
// alguém renomear um campo por engano, é este teste que avisa primeiro.
func TestWeatherForecastPayload_JSON_MatchesReadmeContract(t *testing.T) {
	got, err := json.Marshal(NewWeatherForecastPayload("zone-042", readmeExampleForecast()))
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	want := `{"zoneId":"zone-042","rainProbabilityPercent":80,"forecastTemperatureCelsius":29.5,"forecastWindowHours":12,"source":"open-meteo"}`

	if string(got) != want {
		t.Errorf("JSON payload mismatch.\ngot:  %s\nwant: %s", got, want)
	}
}
