namespace FarmManagement.Domain.DomainEvents;

/// <summary>
/// Raised when a new cultivation zone is registered in the farm.
/// Maps to the "zone.events.v1" Kafka topic (see docs - Event Storming).
/// </summary>
public sealed record ZoneRegistered(
    Guid EventId,
    DateTime OccurredAt,
    string ZoneId,
    string Name,
    decimal Hectares,
    string CropType,
    string SoilType) : IDomainEvent;
