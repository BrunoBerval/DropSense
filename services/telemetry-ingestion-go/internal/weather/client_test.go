package weather

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *OpenMeteoClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := NewOpenMeteoClient()
	client.baseURL = server.URL
	return client
}

// sampleOpenMeteoResponse monta um corpo de resposta no mesmo formato
// real do Open-Meteo (séries horárias paralelas), só com os dois
// campos que este cliente lê.
func sampleOpenMeteoResponse(temps []float64, rain []int) string {
	var body struct {
		Hourly struct {
			Time                     []string  `json:"time"`
			Temperature2m            []float64 `json:"temperature_2m"`
			PrecipitationProbability []int     `json:"precipitation_probability"`
		} `json:"hourly"`
	}
	body.Hourly.Temperature2m = temps
	body.Hourly.PrecipitationProbability = rain
	for range temps {
		body.Hourly.Time = append(body.Hourly.Time, "2026-06-26T00:00")
	}
	data, _ := json.Marshal(body)
	return string(data)
}

func testZone() Zone {
	return Zone{ID: "zone-042", Latitude: -21.37, Longitude: -45.29}
}

func TestFetchForecast_UsesMaxRainProbabilityWithinWindow(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleOpenMeteoResponse(
			[]float64{20, 21, 22, 23},
			[]int{10, 80, 30, 5},
		)))
	})

	forecast, err := client.FetchForecast(context.Background(), testZone(), 4)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// MAX, não média: para decisão de irrigação, uma hora de alta
	// probabilidade no meio de horas secas ainda é motivo para
	// reconsiderar irrigar.
	if forecast.RainProbabilityPercent != 80 {
		t.Errorf("RainProbabilityPercent = %d, want 80 (o máximo, não a média)", forecast.RainProbabilityPercent)
	}
}

func TestFetchForecast_UsesAverageTemperatureWithinWindow(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleOpenMeteoResponse(
			[]float64{20, 24}, // média = 22
			[]int{10, 10},
		)))
	})

	forecast, err := client.FetchForecast(context.Background(), testZone(), 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if forecast.ForecastTemperatureCelsius != 22 {
		t.Errorf("ForecastTemperatureCelsius = %v, want 22", forecast.ForecastTemperatureCelsius)
	}
}

func TestFetchForecast_WindowLargerThanAvailableData_UsesWhatExistsAndReportsIt(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleOpenMeteoResponse(
			[]float64{10, 30}, // só 2 horas disponíveis
			[]int{0, 100},
		)))
	})

	// pede 12h, mas só existem 2 - não deve estourar índice, dar erro,
	// nem fingir que agregou 12h: o WindowHours do resultado deve
	// refletir o que foi de fato usado (honestidade do contrato
	// publicado), não o que foi pedido.
	forecast, err := client.FetchForecast(context.Background(), testZone(), 12)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if forecast.WindowHours != 2 {
		t.Errorf("WindowHours = %d, want 2 (quantidade real disponível, não a pedida)", forecast.WindowHours)
	}
	if forecast.RainProbabilityPercent != 100 {
		t.Errorf("RainProbabilityPercent = %d, want 100", forecast.RainProbabilityPercent)
	}
	if forecast.ForecastTemperatureCelsius != 20 {
		t.Errorf("ForecastTemperatureCelsius = %v, want 20", forecast.ForecastTemperatureCelsius)
	}
}

func TestFetchForecast_SendsZoneCoordinatesAsQueryParams(t *testing.T) {
	var gotLat, gotLon string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotLat = r.URL.Query().Get("latitude")
		gotLon = r.URL.Query().Get("longitude")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleOpenMeteoResponse([]float64{20}, []int{0})))
	})

	if _, err := client.FetchForecast(context.Background(), testZone(), 1); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gotLat == "" || gotLon == "" {
		t.Fatalf("expected latitude/longitude query params, got lat=%q lon=%q", gotLat, gotLon)
	}
}

func TestFetchForecast_WhenServerReturnsNon200_ReturnsError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if _, err := client.FetchForecast(context.Background(), testZone(), 12); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestFetchForecast_WhenResponseHasNoHourlyData_ReturnsError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hourly":{"time":[],"temperature_2m":[],"precipitation_probability":[]}}`))
	})

	if _, err := client.FetchForecast(context.Background(), testZone(), 12); err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestFetchForecast_WhenResponseIsMalformedJSON_ReturnsError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json`))
	})

	if _, err := client.FetchForecast(context.Background(), testZone(), 12); err == nil {
		t.Fatal("expected an error, got nil")
	}
}
