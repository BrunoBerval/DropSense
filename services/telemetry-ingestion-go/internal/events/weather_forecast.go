package events

import (
	"dropsense/telemetry-ingestion/internal/weather"
)

// WeatherForecastEventType, WeatherForecastEventVersion e
// WeatherForecastTopic identificam o evento WeatherForecastUpdated,
// publicado no tópico weather.forecasts.v1, conforme o contrato de
// eventos do README ("EVENT CONTRACTS - SOURCE OF TRUTH" > Telemetry
// Ingestion Events).
const (
	WeatherForecastEventType    = "WeatherForecastUpdated"
	WeatherForecastEventVersion = 1
	WeatherForecastTopic        = "weather.forecasts.v1"

	// weatherForecastSource é sempre "open-meteo" hoje: é a única
	// fonte que este serviço consulta (ver README, Parte 2: "consulta
	// uma API externa de previsão do tempo").
	weatherForecastSource = "open-meteo"
)

// WeatherForecastPayload é o formato de FIO do evento
// WeatherForecastUpdated - mesmo raciocínio de SoilReadingPayload:
// struct separada do domínio (weather.Forecast), com os nomes JSON
// exatos do contrato publicado no README.
type WeatherForecastPayload struct {
	ZoneID                     string  `json:"zoneId"`
	RainProbabilityPercent     int     `json:"rainProbabilityPercent"`
	ForecastTemperatureCelsius float64 `json:"forecastTemperatureCelsius"`
	ForecastWindowHours        int     `json:"forecastWindowHours"`
	Source                     string  `json:"source"`
}

// NewWeatherForecastPayload traduz um forecast já agregado (ver
// weather.Forecast) para o formato de fio do evento. zoneID vem de
// fora porque weather.Forecast não carrega identidade de zona - é só
// o resultado numérico agregado.
func NewWeatherForecastPayload(zoneID string, forecast weather.Forecast) WeatherForecastPayload {
	return WeatherForecastPayload{
		ZoneID:                     zoneID,
		RainProbabilityPercent:     forecast.RainProbabilityPercent,
		ForecastTemperatureCelsius: forecast.ForecastTemperatureCelsius,
		ForecastWindowHours:        forecast.WindowHours,
		Source:                     weatherForecastSource,
	}
}
