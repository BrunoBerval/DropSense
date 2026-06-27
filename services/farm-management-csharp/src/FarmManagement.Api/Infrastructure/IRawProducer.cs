namespace FarmManagement.Api.Infrastructure;

// IRawProducer/IRawConsumer existem pelo mesmo motivo de produceClient
// (internal/kafka/producer.go) e do parâmetro `client` injetável em
// TopicConsumer (app/kafka_consumer.py): uma fatia mínima do que este
// serviço realmente usa do Confluent.Kafka, para os testes poderem
// trocar por um fake e nunca precisar de um broker real durante
// "dotnet test".
public interface IRawProducer
{
    void Produce(string topic, string key, string value);
}

public interface IRawConsumer
{
    // Devolve null se nada chegou dentro do timeout - mesmo
    // contrato de Consumer.poll() (confluent-kafka-python) e
    // ProduceSync (franz-go): ausência de mensagem não é erro.
    string? ConsumeValue(TimeSpan timeout);

    void Close();
}
