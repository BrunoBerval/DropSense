using Confluent.Kafka;

namespace FarmManagement.Api.Infrastructure;

// Implementação real de IRawProducer sobre o client oficial da
// Confluent - mesmo fornecedor do confluent-kafka-python já usado no
// irrigation-prediction, mantendo o mesmo vendor nas três linguagens
// do projeto.
public sealed class ConfluentRawProducer : IRawProducer, IDisposable
{
    private readonly IProducer<string, string> _producer;

    public ConfluentRawProducer(string brokers)
    {
        _producer = new ProducerBuilder<string, string>(new ProducerConfig { BootstrapServers = brokers }).Build();
    }

    public void Produce(string topic, string key, string value)
    {
        // Fire-and-forget com callback de erro logado - mesma postura
        // de "loga e segue" do Pipeline.worker (Go) e do
        // DecisionScheduler (Python): uma falha de publish não deve
        // derrubar o worker que está processando o restante da fila.
        _producer.Produce(topic, new Message<string, string> { Key = key, Value = value }, report =>
        {
            if (report.Error.IsError)
            {
                Console.Error.WriteLine($"failed to publish to {topic}: {report.Error.Reason}");
            }
        });
    }

    public void Dispose() => _producer.Flush(TimeSpan.FromSeconds(5));
}

// Implementação real de IRawConsumer. ConsumeValue devolve só o valor
// (string) da mensagem - quem chama (DecisionConsumerWorker) decide o
// que fazer com isso; esta classe não sabe nada sobre envelopes ou
// payloads, só sobre transporte.
public sealed class ConfluentRawConsumer : IRawConsumer, IDisposable
{
    private readonly IConsumer<Ignore, string> _consumer;

    public ConfluentRawConsumer(string brokers, string groupId, string topic)
    {
        _consumer = new ConsumerBuilder<Ignore, string>(new ConsumerConfig
        {
            BootstrapServers = brokers,
            GroupId = groupId,
            AutoOffsetReset = AutoOffsetReset.Earliest,
        }).Build();

        _consumer.Subscribe(topic);
    }

    public string? ConsumeValue(TimeSpan timeout)
    {
        var result = _consumer.Consume(timeout);
        return result?.Message?.Value;
    }

    public void Close() => _consumer.Close();

    public void Dispose() => _consumer.Dispose();
}
