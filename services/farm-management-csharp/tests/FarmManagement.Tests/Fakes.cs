using FarmManagement.Api.Application;
using FarmManagement.Api.Domain;
using FarmManagement.Api.Infrastructure;

namespace FarmManagement.Tests;

// Mesmo papel de fakeSubmitter/recordingPublisher (Go) e
// fakePublisher/fakeProduceClient (Python): dublês simples, sem
// framework de mock, para testar a lógica sem broker nenhum.

public sealed class FakeZoneRepository : IZoneRepository
{
    public Dictionary<string, Zone> Zones { get; } = new();

    public Zone? Find(string zoneId) => Zones.TryGetValue(zoneId, out var zone) ? zone : null;
}

public sealed class FakePublisher : IIrrigationEventPublisher
{
    public List<(string ZoneId, string CorrelationId)> Started { get; } = new();
    public List<(string ZoneId, RejectionReason Reason, string CorrelationId)> Rejected { get; } = new();

    public Task PublishIrrigationStartedAsync(string zoneId, string correlationId)
    {
        Started.Add((zoneId, correlationId));
        return Task.CompletedTask;
    }

    public Task PublishIrrigationRejectedAsync(string zoneId, RejectionReason reason, string correlationId)
    {
        Rejected.Add((zoneId, reason, correlationId));
        return Task.CompletedTask;
    }
}

public sealed class FakeRawProducer : IRawProducer
{
    public List<(string Topic, string Key, string Value)> Calls { get; } = new();

    public void Produce(string topic, string key, string value) => Calls.Add((topic, key, value));
}
