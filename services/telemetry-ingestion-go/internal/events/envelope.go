package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Envelope é o formato padrão de evento publicado no Kafka pelo
// DropSense, conforme definido no README ("EVENT CONTRACTS - SOURCE OF
// TRUTH" > Event Envelope). Todo produtor de evento (hoje só o Go;
// futuramente Python e C# também) embrulha o payload específico daquele
// evento dentro de um Envelope antes de publicar.
//
// É a mesma ideia de Anti-Corruption Layer do handler.go (readingPayload
// vs. ingestion.SoilReading), só que na borda de SAÍDA em vez de entrada:
// aqui não confundimos a forma como o evento existe no nosso domínio
// (ingestion.SoilReading) com a forma como ele é publicado para o resto
// do sistema (este envelope + um payload de fio próprio, como
// SoilReadingPayload em soil_reading.go).
type Envelope struct {
	EventID       string          `json:"eventId"`
	EventType     string          `json:"eventType"`
	EventVersion  int             `json:"eventVersion"`
	OccurredAt    string          `json:"occurredAt"`
	Producer      string          `json:"producer"`
	CorrelationID string          `json:"correlationId"`
	Payload       json.RawMessage `json:"payload"`
}

// NewEnvelope monta o envelope em volta de um payload já no formato de
// fio (ex.: SoilReadingPayload). eventId e occurredAt são gerados aqui,
// na borda de publicação - não são decisão de negócio do chamador.
// occurredAt usa o mesmo formato (RFC3339, sem fração de segundo) que o
// resto do contrato de fio do projeto já usa para timestamps (ver
// readingPayload.MeasuredAt em handler.go) - consistência com o que já
// existe, e bate exatamente com o exemplo do README
// ("2026-06-23T14:32:10Z").
//
// correlationID é responsabilidade de quem chama: para um evento raiz,
// sem nada antes dele na cadeia (como o SoilReadingRegistered), o
// chamador gera um novo id; para eventos derivados de uma cadeia já em
// andamento (ex.: IrrigationStarted depois de IrrigationDecisionCalculated),
// o chamador propaga o correlationId que já recebeu.
func NewEnvelope(eventType string, eventVersion int, producer, correlationID string, payload any) (Envelope, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		EventID:       uuid.NewString(),
		EventType:     eventType,
		EventVersion:  eventVersion,
		OccurredAt:    time.Now().UTC().Format(time.RFC3339),
		Producer:      producer,
		CorrelationID: correlationID,
		Payload:       rawPayload,
	}, nil
}
