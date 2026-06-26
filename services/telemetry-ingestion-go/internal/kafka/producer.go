package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"dropsense/telemetry-ingestion/internal/events"
	"dropsense/telemetry-ingestion/internal/ingestion"
)

// produceClient é o subconjunto de *kgo.Client que o Producer realmente
// usa. Existir como interface, em vez de o Producer guardar um
// *kgo.Client direto, é o mesmo motivo da interface Submitter em
// handler.go: permite testar a montagem da mensagem (envelope, chave de
// partição, tópico) sem precisar de um broker Kafka real de pé durante
// "go test".
type produceClient interface {
	ProduceSync(ctx context.Context, rs ...*kgo.Record) kgo.ProduceResults
	Close()
}

// Producer implementa ingestion.Publisher publicando de fato no Kafka -
// é a peça que faltava, anunciada no comentário de publisher.go ("the
// real implementation... doesn't exist yet; it will live in a future
// internal/kafka package").
type Producer struct {
	client produceClient
	topic  string
}

// NewProducer abre uma conexão real com o(s) broker(s) informado(s) e
// devolve um Producer pronto para publicar no tópico indicado. brokers
// segue o formato "host:porta" (ex.: "kafka:9092", o nome do serviço
// dentro da rede do docker-compose).
func NewProducer(brokers []string, topic string) (*Producer, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to create client: %w", err)
	}
	return &Producer{client: client, topic: topic}, nil
}

// Publish traduz a leitura de domínio para o formato de fio
// (events.SoilReadingPayload), embrulha no envelope padrão definido no
// README, e publica usando o sensorId como chave de partição - é essa
// chave que garante que todas as leituras de um mesmo sensor caem na
// mesma partição e, portanto, mantêm ordem entre si (conforme a tabela
// de tópicos do README: partition key = sensorId).
//
// O correlationId é gerado aqui porque SoilReadingRegistered é um evento
// raiz: nada "antes" dele inicia essa cadeia de correlação (diferente de
// IrrigationStarted/IrrigationFinished, que propagam o correlationId
// recebido de IrrigationDecisionCalculated).
func (p *Producer) Publish(ctx context.Context, reading ingestion.SoilReading) error {
	payload := events.NewSoilReadingPayload(reading)

	envelope, err := events.NewEnvelope(
		events.SoilReadingEventType,
		events.SoilReadingEventVersion,
		"telemetry-ingestion",
		uuid.NewString(),
		payload,
	)
	if err != nil {
		return fmt.Errorf("kafka: failed to build envelope: %w", err)
	}

	value, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("kafka: failed to marshal envelope: %w", err)
	}

	record := &kgo.Record{
		Topic: p.topic,
		Key:   []byte(reading.SensorID),
		Value: value,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("kafka: failed to publish reading from sensor %s: %w", reading.SensorID, err)
	}
	return nil
}

// Close libera a conexão com o broker. Chamado uma vez, no shutdown
// gracioso do main().
func (p *Producer) Close() error {
	p.client.Close()
	return nil
}
