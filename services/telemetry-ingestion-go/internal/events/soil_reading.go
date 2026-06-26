package events

import (
	"time"

	"dropsense/telemetry-ingestion/internal/ingestion"
)

// SoilReadingEventType e SoilReadingEventVersion identificam o evento
// SoilReadingRegistered, publicado no tópico telemetry.readings.v1,
// conforme o contrato de eventos do README ("EVENT CONTRACTS - SOURCE OF
// TRUTH" > Telemetry Ingestion Events).
const (
	SoilReadingEventType    = "SoilReadingRegistered"
	SoilReadingEventVersion = 1
	// SoilReadingTopic conforme a tabela de tópicos do README. Antes
	// vinha de fora (KAFKA_TOPIC, env var) porque o Producer só
	// publicava num tópico só; agora que ele publica em mais de um
	// (telemetry.readings.v1 e weather.forecasts.v1), o tópico passou
	// a ser parte do contrato de cada evento, não um detalhe de
	// deploy.
	SoilReadingTopic = "telemetry.readings.v1"
)

// SoilReadingPayload é o formato de FIO do evento SoilReadingRegistered -
// os mesmos cinco campos, com os mesmos nomes JSON, documentados no
// README. É deliberadamente uma struct separada de ingestion.SoilReading:
// o domínio usa nomes Go-idiomáticos (PascalCase, time.Time); o contrato
// publicado usa camelCase e timestamp como string RFC3339. Sem essa
// separação, json.Marshal no SoilReading direto vazaria a convenção de
// nomenclatura do Go (ex.: "SensorID" em vez de "sensorId") para o resto
// do sistema - quebrando o contrato que Python e C# vão depender. É o
// mesmo raciocínio de readingPayload em handler.go, espelhado na borda
// de saída.
type SoilReadingPayload struct {
	SensorID               string  `json:"sensorId"`
	ZoneID                 string  `json:"zoneId"`
	SoilMoisturePercent    float64 `json:"soilMoisturePercent"`
	SoilTemperatureCelsius float64 `json:"soilTemperatureCelsius"`
	MeasuredAt             string  `json:"measuredAt"`
}

// NewSoilReadingPayload traduz a leitura já validada do domínio para o
// formato de fio do evento. measuredAt é o instante da MEDIÇÃO (vem do
// sensor) - diferente do occurredAt do envelope, que é o instante da
// PUBLICAÇÃO. O README faz essa distinção explicitamente nas notas do
// SoilReadingRegistered.
func NewSoilReadingPayload(reading ingestion.SoilReading) SoilReadingPayload {
	return SoilReadingPayload{
		SensorID:               reading.SensorID,
		ZoneID:                 reading.ZoneID,
		SoilMoisturePercent:    reading.SoilMoisturePercent,
		SoilTemperatureCelsius: reading.SoilTemperatureCelsius,
		MeasuredAt:             reading.MeasuredAt.UTC().Format(time.RFC3339),
	}
}
