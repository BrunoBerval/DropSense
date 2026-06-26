package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// defaultBaseURL é o endpoint público do Open-Meteo, conforme já
// referenciado no exemplo de WeatherForecastUpdated do README
// ("source": "open-meteo"). Gratuito, sem API key - coerente com a
// filosofia de "rodar o mínimo necessário" já adotada no resto da
// infraestrutura deste projeto.
const defaultBaseURL = "https://api.open-meteo.com/v1/forecast"

// Forecast é o resultado já agregado para uma janela de horas - a
// forma "pronta para publicar", depois de processar a resposta crua
// do Open-Meteo (que vem como uma série horária, não como um único
// número). WindowHours reflete a quantidade REAL de horas agregadas,
// que pode ser menor do que a pedida se a API devolver menos dados do
// que isso (ver aggregate).
type Forecast struct {
	WindowHours                int
	RainProbabilityPercent     int
	ForecastTemperatureCelsius float64
}

// openMeteoResponse mapeia só os campos que este serviço usa da
// resposta do Open-Meteo. A API devolve muito mais do que isso, mas
// não há motivo para modelar o que não é consumido.
type openMeteoResponse struct {
	Hourly struct {
		Temperature2m            []float64 `json:"temperature_2m"`
		PrecipitationProbability []int     `json:"precipitation_probability"`
	} `json:"hourly"`
}

// OpenMeteoClient busca previsão do tempo na API pública do
// Open-Meteo. httpClient e baseURL são configuráveis (este último,
// não exportado) para permitir testes com httptest.Server, sem bater
// na API real durante "go test".
type OpenMeteoClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewOpenMeteoClient devolve um client pronto para uso contra a API
// real do Open-Meteo.
func NewOpenMeteoClient() *OpenMeteoClient {
	return &OpenMeteoClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    defaultBaseURL,
	}
}

// FetchForecast busca a previsão horária da zona e agrega para a
// janela pedida (windowHours), olhando as primeiras windowHours
// entradas retornadas pelo Open-Meteo a partir de agora.
//
// rainProbabilityPercent usa o MÁXIMO da janela, de propósito: para
// decisão de irrigação, a pergunta que importa é "existe uma chance
// relevante de chover nessa janela", não a média - uma hora de alta
// probabilidade no meio de horas secas ainda é motivo para
// reconsiderar irrigar. forecastTemperatureCelsius usa a MÉDIA, por
// representar melhor "como vai estar o clima" ao longo da janela como
// um todo.
func (c *OpenMeteoClient) FetchForecast(ctx context.Context, zone Zone, windowHours int) (Forecast, error) {
	url := fmt.Sprintf(
		"%s?latitude=%f&longitude=%f&hourly=temperature_2m,precipitation_probability&timezone=UTC&forecast_days=2",
		c.baseURL, zone.Latitude, zone.Longitude,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Forecast{}, fmt.Errorf("weather: failed to build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Forecast{}, fmt.Errorf("weather: failed to call open-meteo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Forecast{}, fmt.Errorf("weather: open-meteo returned status %d", resp.StatusCode)
	}

	var parsed openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Forecast{}, fmt.Errorf("weather: failed to decode open-meteo response: %w", err)
	}

	return aggregate(parsed, windowHours)
}

func aggregate(resp openMeteoResponse, windowHours int) (Forecast, error) {
	available := len(resp.Hourly.Temperature2m)
	if available == 0 || len(resp.Hourly.PrecipitationProbability) == 0 {
		return Forecast{}, fmt.Errorf("weather: open-meteo response has no hourly data")
	}

	n := windowHours
	if n <= 0 || n > available {
		n = available
	}

	maxRain := 0
	var tempSum float64
	for i := 0; i < n; i++ {
		if resp.Hourly.PrecipitationProbability[i] > maxRain {
			maxRain = resp.Hourly.PrecipitationProbability[i]
		}
		tempSum += resp.Hourly.Temperature2m[i]
	}

	return Forecast{
		WindowHours:                n,
		RainProbabilityPercent:     maxRain,
		ForecastTemperatureCelsius: tempSum / float64(n),
	}, nil
}
