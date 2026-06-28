package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"dropsense/mock-sensors/internal/csvsource"
)

// httpDoer é o subconjunto de *http.Client que o Client realmente
// usa - mesmo motivo de produceClient (internal/kafka, Go),
// IRawProducer (C#) e do client injetável em TopicConsumer (Python):
// permite testar a montagem da requisição sem precisar de um
// servidor real durante "go test".
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// wirePayload espelha exatamente readingPayload em handler.go do
// telemetry-ingestion - é o outro lado do mesmo contrato.
type wirePayload struct {
	SensorID               string  `json:"sensorId"`
	ZoneID                 string  `json:"zoneId"`
	SoilMoisturePercent    float64 `json:"soilMoisturePercent"`
	SoilTemperatureCelsius float64 `json:"soilTemperatureCelsius"`
	MeasuredAt             string  `json:"measuredAt"`
}

// Client envia uma leitura para o /readings do telemetry-ingestion.
type Client struct {
	httpClient httpDoer
	baseURL    string
}

func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,
	}
}

// Send carimba measuredAt com o instante REAL do envio - é aqui, e
// só aqui, que "o horário é definido pelo job do server", exatamente
// como combinamos: o CSV nunca carrega timestamp, só os valores.
func (c *Client) Send(ctx context.Context, reading csvsource.Reading) error {
	payload := wirePayload{
		SensorID:               reading.SensorID,
		ZoneID:                 reading.ZoneID,
		SoilMoisturePercent:    reading.SoilMoisturePercent,
		SoilTemperatureCelsius: reading.SoilTemperatureCelsius,
		MeasuredAt:             time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sender: failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/readings", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sender: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sender: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("sender: unexpected status %d from sensor %s", resp.StatusCode, reading.SensorID)
	}
	return nil
}
