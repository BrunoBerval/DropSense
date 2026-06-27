using System.Text.Json;
using FarmManagement.Api.Events;

namespace FarmManagement.Tests;

public class EnvelopeTests
{
    [Fact]
    public void Wrap_SetsCallerProvidedFields()
    {
        var payload = new IrrigationStartedPayload { ZoneId = "zone-042", CycleId = "c1", StartedAt = "2026-06-23T14:31:05Z" };

        var envelope = Envelope<IrrigationStartedPayload>.Wrap(EventTypes.IrrigationStarted, "corr-123", payload);

        Assert.Equal("IrrigationStarted", envelope.EventType);
        Assert.Equal("corr-123", envelope.CorrelationId);
        Assert.Equal("farm-management", envelope.Producer);
        Assert.Equal(EventTypes.CurrentVersion, envelope.EventVersion);
    }

    [Fact]
    public void Wrap_GeneratesValidGuidEventId()
    {
        var payload = new IrrigationStartedPayload { ZoneId = "zone-042", CycleId = "c1", StartedAt = "2026-06-23T14:31:05Z" };

        var envelope = Envelope<IrrigationStartedPayload>.Wrap(EventTypes.IrrigationStarted, "corr-123", payload);

        Assert.True(Guid.TryParse(envelope.EventId, out _));
    }

    [Fact]
    public void Wrap_TwoCalls_ProduceDifferentEventIds()
    {
        var payload = new IrrigationStartedPayload { ZoneId = "zone-042", CycleId = "c1", StartedAt = "2026-06-23T14:31:05Z" };

        var first = Envelope<IrrigationStartedPayload>.Wrap(EventTypes.IrrigationStarted, "corr-123", payload);
        var second = Envelope<IrrigationStartedPayload>.Wrap(EventTypes.IrrigationStarted, "corr-123", payload);

        Assert.NotEqual(first.EventId, second.EventId);
    }

    [Fact]
    public void Wrap_OccurredAt_IsRfc3339AndRecent()
    {
        var payload = new IrrigationStartedPayload { ZoneId = "zone-042", CycleId = "c1", StartedAt = "2026-06-23T14:31:05Z" };

        var before = DateTime.UtcNow;
        var envelope = Envelope<IrrigationStartedPayload>.Wrap(EventTypes.IrrigationStarted, "corr-123", payload);
        var after = DateTime.UtcNow;

        var occurredAt = DateTime.Parse(envelope.OccurredAt.TrimEnd('Z')).ToUniversalTime();

        Assert.InRange(occurredAt, before.AddSeconds(-2), after.AddSeconds(2));
    }

    [Fact]
    public void Json_UsesContractFieldNames()
    {
        var payload = new IrrigationStartedPayload { ZoneId = "zone-042", CycleId = "c1", StartedAt = "2026-06-23T14:31:05Z" };
        var envelope = Envelope<IrrigationStartedPayload>.Wrap(EventTypes.IrrigationStarted, "corr-123", payload);

        var json = JsonSerializer.Serialize(envelope);
        var asDict = JsonSerializer.Deserialize<Dictionary<string, object>>(json)!;

        foreach (var field in new[] { "eventId", "eventType", "eventVersion", "occurredAt", "producer", "correlationId", "payload" })
        {
            Assert.True(asDict.ContainsKey(field), $"expected field '{field}' in envelope JSON");
        }
    }

    [Fact]
    public void Deserialize_ParsesARealCapturedMessageFromTheRunningStack()
    {
        // Esta não é uma mensagem inventada - é, literalmente, o que
        // apareceu no kafka-console-consumer durante a validação
        // manual da fase anterior (irrigation-prediction publicando
        // de verdade). Comparar contra dado real, não só um exemplo
        // do README, é o que prova que este serviço entende o que o
        // Python publica de fato, não só o que o contrato documenta.
        const string raw = """
            {"eventId": "e2c97860-8445-4cdd-8917-8b71302e9437", "eventType": "IrrigationDecisionCalculated", "eventVersion": 1, "occurredAt": "2026-06-26T23:43:01Z", "producer": "irrigation-prediction", "correlationId": "e01d624d-ae6e-4779-a4b1-351f841d6dad", "payload": {"zoneId": "zone-042", "decision": "SKIP_IRRIGATION", "windowStart": "2026-06-26T23:38:01Z", "windowEnd": "2026-06-26T23:43:01Z", "averageSoilMoisturePercent": 38.5, "rainProbabilityPercent": 0, "confidenceScore": 0.9, "modelVersion": "v1"}}
            """;

        var envelope = JsonSerializer.Deserialize<Envelope<IrrigationDecisionPayload>>(raw)!;

        Assert.Equal("IrrigationDecisionCalculated", envelope.EventType);
        Assert.Equal("e01d624d-ae6e-4779-a4b1-351f841d6dad", envelope.CorrelationId);
        Assert.Equal("zone-042", envelope.Payload.ZoneId);
        Assert.Equal("SKIP_IRRIGATION", envelope.Payload.Decision);
        Assert.Equal(38.5, envelope.Payload.AverageSoilMoisturePercent);
    }
}
