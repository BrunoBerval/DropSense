using System.Text.Json;
using FarmManagement.Api.Domain;
using FarmManagement.Api.Events;
using FarmManagement.Api.Infrastructure;

namespace FarmManagement.Tests;

public class KafkaIrrigationEventPublisherTests
{
    [Fact]
    public async Task PublishIrrigationStartedAsync_SendsToIrrigationEventsTopic()
    {
        var producer = new FakeRawProducer();
        var publisher = new KafkaIrrigationEventPublisher(producer);

        await publisher.PublishIrrigationStartedAsync("zone-042", "corr-999");

        Assert.Single(producer.Calls);
        Assert.Equal(Topics.IrrigationEvents, producer.Calls[0].Topic);
    }

    [Fact]
    public async Task PublishIrrigationStartedAsync_UsesZoneIdAsPartitionKey()
    {
        var producer = new FakeRawProducer();
        var publisher = new KafkaIrrigationEventPublisher(producer);

        await publisher.PublishIrrigationStartedAsync("zone-042", "corr-999");

        Assert.Equal("zone-042", producer.Calls[0].Key);
    }

    [Fact]
    public async Task PublishIrrigationStartedAsync_PropagatesCorrelationId_DoesNotRegenerate()
    {
        var producer = new FakeRawProducer();
        var publisher = new KafkaIrrigationEventPublisher(producer);

        await publisher.PublishIrrigationStartedAsync("zone-042", "corr-999");

        var envelope = JsonSerializer.Deserialize<Envelope<IrrigationStartedPayload>>(producer.Calls[0].Value)!;
        Assert.Equal("corr-999", envelope.CorrelationId);
        Assert.Equal("IrrigationStarted", envelope.EventType);
        Assert.False(string.IsNullOrEmpty(envelope.Payload.CycleId));
    }

    [Fact]
    public async Task PublishIrrigationRejectedAsync_PayloadHasContractReasonString()
    {
        var producer = new FakeRawProducer();
        var publisher = new KafkaIrrigationEventPublisher(producer);

        await publisher.PublishIrrigationRejectedAsync("zone-042", RejectionReason.ReservoirInsufficientVolume, "corr-888");

        var envelope = JsonSerializer.Deserialize<Envelope<IrrigationRejectedPayload>>(producer.Calls[0].Value)!;
        Assert.Equal("IrrigationRejected", envelope.EventType);
        Assert.Equal("RESERVOIR_INSUFFICIENT_VOLUME", envelope.Payload.Reason);
    }
}
