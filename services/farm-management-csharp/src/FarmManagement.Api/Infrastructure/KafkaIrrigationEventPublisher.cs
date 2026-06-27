using System.Text.Json;
using FarmManagement.Api.Application;
using FarmManagement.Api.Domain;
using FarmManagement.Api.Events;

namespace FarmManagement.Api.Infrastructure;

// Implementação real de IIrrigationEventPublisher: monta o envelope,
// serializa, e publica via IRawProducer - mesmo papel de
// internal/kafka.Producer no Go e DecisionPublisher no Python.
public sealed class KafkaIrrigationEventPublisher : IIrrigationEventPublisher
{
    private readonly IRawProducer _producer;

    public KafkaIrrigationEventPublisher(IRawProducer producer)
    {
        _producer = producer;
    }

    public Task PublishIrrigationStartedAsync(string zoneId, string correlationId)
    {
        var payload = new IrrigationStartedPayload
        {
            ZoneId = zoneId,
            CycleId = Guid.NewGuid().ToString(),
            StartedAt = DateTime.UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ"),
        };

        Publish(EventTypes.IrrigationStarted, zoneId, correlationId, payload);
        return Task.CompletedTask;
    }

    public Task PublishIrrigationRejectedAsync(string zoneId, RejectionReason reason, string correlationId)
    {
        var payload = new IrrigationRejectedPayload
        {
            ZoneId = zoneId,
            Reason = reason.ToContractString(),
            RejectedAt = DateTime.UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ"),
        };

        Publish(EventTypes.IrrigationRejected, zoneId, correlationId, payload);
        return Task.CompletedTask;
    }

    // zoneId é a chave de partição em irrigation.events.v1, conforme
    // a tabela de tópicos do README - mesma coluna que os outros
    // eventos dessa zona já usam nos outros serviços.
    private void Publish<TPayload>(string eventType, string zoneId, string correlationId, TPayload payload)
    {
        var envelope = Envelope<TPayload>.Wrap(eventType, correlationId, payload);
        var value = JsonSerializer.Serialize(envelope);
        _producer.Produce(Topics.IrrigationEvents, zoneId, value);
    }
}
