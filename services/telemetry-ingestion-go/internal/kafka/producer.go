package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"dropsense/telemetry-ingestion/internal/events"
	"dropsense/telemetry-ingestion/internal/ingestion"
	"dropsense/telemetry-ingestion/internal/weather"
)

// producerName identifica este serviço no campo "producer" do
// envelope - mesmo valor em todo evento que este serviço publica,
// conforme o contrato do README.
const producerName = "telemetry-ingestion"

// produceClient é o subconjunto de *kgo.Client que o Producer
// realmente usa. Existir como interface, em vez de o Producer guardar
// um *kgo.Client direto, é o mesmo motivo da interface Submitter em
// handler.go: permite testar a montagem da mensagem sem precisar de
// um broker Kafka real de pé durante "go test".
type produceClient interface {
	ProduceSync(ctx context.Context, rs ...*kgo.Record) kgo.ProduceResults
	Close()
}

// Producer implementa duas interfaces com a mesma implementação:
// ingestion.Publisher (para leituras de solo) e
// weather.ForecastPublisher (para previsão do tempo). Ambas são, no
// fim, "montar um envelope e publicar no Kafka" - só o tópico, a
// chave de partição e o payload mudam.
type Producer struct {
	client produceClient
}

// NewProducer abre uma conexão real com o(s) broker(s) informado(s).
// brokers segue o formato "host:porta" (ex.: "kafka:9092", o nome do
// serviço dentro da rede do docker-compose).
func NewProducer(brokers []string) (*Producer, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to create client: %w", err)
	}
	return &Producer{client: client}, nil
}

// Publish traduz a leitura de domínio para o formato de fio
// (events.SoilReadingPayload), embrulha no envelope padrão, e publica
// em telemetry.readings.v1 usando sensorId como chave de partição -
// conforme a tabela de tópicos do README.
//
// O correlationId é gerado aqui porque SoilReadingRegistered é um
// evento raiz: nada "antes" dele inicia essa cadeia de correlação.
func (p *Producer) Publish(ctx context.Context, reading ingestion.SoilReading) error {
	payload := events.NewSoilReadingPayload(reading)

	envelope, err := events.NewEnvelope(
		events.SoilReadingEventType,
		events.SoilReadingEventVersion,
		producerName,
		uuid.NewString(),
		payload,
	)
	if err != nil {
		return fmt.Errorf("kafka: failed to build envelope: %w", err)
	}

	return p.publishEnvelope(ctx, events.SoilReadingTopic, reading.SensorID, envelope)
}

// PublishWeatherForecast traduz um forecast já agregado para o
// formato de fio (events.WeatherForecastPayload), embrulha no
// envelope padrão, e publica em weather.forecasts.v1 usando zoneId
// como chave de partição - mesma coluna usada pelos outros eventos
// dessa zona na tabela de tópicos do README. Satisfaz
// weather.ForecastPublisher.
func (p *Producer) PublishWeatherForecast(ctx context.Context, zoneID string, forecast weather.Forecast) error {
	payload := events.NewWeatherForecastPayload(zoneID, forecast)

	envelope, err := events.NewEnvelope(
		events.WeatherForecastEventType,
		events.WeatherForecastEventVersion,
		producerName,
		uuid.NewString(),
		payload,
	)
	if err != nil {
		return fmt.Errorf("kafka: failed to build envelope: %w", err)
	}

	return p.publishEnvelope(ctx, events.WeatherForecastTopic, zoneID, envelope)
}

func (p *Producer) publishEnvelope(ctx context.Context, topic, key string, envelope events.Envelope) error {
	value, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("kafka: failed to marshal envelope: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("kafka: failed to publish %s: %w", envelope.EventType, err)
	}
	return nil
}

// Close libera a conexão com o broker. Chamado uma vez, no shutdown
// gracioso do main().
func (p *Producer) Close() error {
	p.client.Close()
	return nil
}
