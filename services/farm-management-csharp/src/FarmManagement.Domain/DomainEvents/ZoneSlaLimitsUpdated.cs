namespace FarmManagement.Domain.DomainEvents;

/// <summary>
/// Raised when the acceptable soil moisture range (SLA) for a zone changes.
/// Go and Python consume this (via "zone.events.v1") to keep their local
/// read-model of zone configuration in sync (Event-Carried State Transfer).
/// </summary>
public sealed record ZoneSlaLimitsUpdated(
    Guid EventId,
    DateTime OccurredAt,
    string ZoneId,
    double MinSoilMoisturePercent,
    double MaxSoilMoisturePercent) : IDomainEvent;
