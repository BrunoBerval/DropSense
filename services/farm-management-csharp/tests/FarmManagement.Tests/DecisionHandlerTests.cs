using FarmManagement.Api.Domain;
using FarmManagement.Api.Application;
using FarmManagement.Api.Events;

namespace FarmManagement.Tests;

public class DecisionHandlerTests
{
    private static Envelope<IrrigationDecisionPayload> MakeDecisionEnvelope(
        string zoneId, string decision, string correlationId = "corr-xyz") =>
        Envelope<IrrigationDecisionPayload>.Wrap(
            EventTypes.IrrigationDecisionCalculated,
            correlationId,
            new IrrigationDecisionPayload
            {
                ZoneId = zoneId,
                Decision = decision,
                WindowStart = "2026-06-23T14:25:00Z",
                WindowEnd = "2026-06-23T14:30:00Z",
                AverageSoilMoisturePercent = 20.0,
                RainProbabilityPercent = 0,
                ConfidenceScore = 0.8,
                ModelVersion = "v1",
            });

    [Fact]
    public async Task HandleAsync_StartIrrigation_ApprovedZone_PublishesStarted()
    {
        var repo = new FakeZoneRepository();
        repo.Zones["zone-042"] = new Zone("zone-042", isUnderMaintenance: false, new Reservoir(true));
        var publisher = new FakePublisher();
        var handler = new DecisionHandler(repo, publisher);

        await handler.HandleAsync(MakeDecisionEnvelope("zone-042", Decisions.StartIrrigation, "corr-abc"));

        Assert.Single(publisher.Started);
        Assert.Equal("corr-abc", publisher.Started[0].CorrelationId);
        Assert.Empty(publisher.Rejected);
    }

    [Fact]
    public async Task HandleAsync_StartIrrigation_ZoneUnderMaintenance_PublishesRejected()
    {
        var repo = new FakeZoneRepository();
        repo.Zones["zone-042"] = new Zone("zone-042", isUnderMaintenance: true, new Reservoir(true));
        var publisher = new FakePublisher();
        var handler = new DecisionHandler(repo, publisher);

        await handler.HandleAsync(MakeDecisionEnvelope("zone-042", Decisions.StartIrrigation));

        Assert.Single(publisher.Rejected);
        Assert.Equal(RejectionReason.ZoneUnderMaintenance, publisher.Rejected[0].Reason);
        Assert.Empty(publisher.Started);
    }

    [Fact]
    public async Task HandleAsync_SkipIrrigation_PublishesNothing()
    {
        // Python já decidiu não irrigar - não há nada para este
        // serviço validar ou rejeitar.
        var repo = new FakeZoneRepository();
        repo.Zones["zone-042"] = new Zone("zone-042", isUnderMaintenance: false, new Reservoir(true));
        var publisher = new FakePublisher();
        var handler = new DecisionHandler(repo, publisher);

        await handler.HandleAsync(MakeDecisionEnvelope("zone-042", Decisions.SkipIrrigation));

        Assert.Empty(publisher.Started);
        Assert.Empty(publisher.Rejected);
    }

    [Fact]
    public async Task HandleAsync_UnknownZone_PublishesNothing_DoesNotThrow()
    {
        var repo = new FakeZoneRepository(); // vazio
        var publisher = new FakePublisher();
        var handler = new DecisionHandler(repo, publisher);

        await handler.HandleAsync(MakeDecisionEnvelope("zone-desconhecida", Decisions.StartIrrigation));

        Assert.Empty(publisher.Started);
        Assert.Empty(publisher.Rejected);
    }
}
