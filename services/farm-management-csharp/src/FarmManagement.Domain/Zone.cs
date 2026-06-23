using FarmManagement.Domain.DomainEvents;
using FarmManagement.Domain.Exceptions;

namespace FarmManagement.Domain;

/// <summary>
/// Aggregate root representing a cultivation zone within the farm.
/// Owns the invariants around its own registration and SLA configuration.
///
/// Decisions that live *outside* this aggregate on purpose:
/// - Whether to start/stop irrigation (that's the IrrigationCycle aggregate,
///   which also checks Reservoir state - not built yet).
/// - Whether a reading currently breaches the SLA (that belongs to whatever
///   process evaluates incoming readings against MinSoilMoisturePercent /
///   MaxSoilMoisturePercent and raises SlaBreached - not built yet).
/// Zone's only job is to hold its own configuration correctly.
/// </summary>
public sealed class Zone
{
    private readonly List<IDomainEvent> _domainEvents = new();

    public string ZoneId { get; }
    public string Name { get; }
    public decimal Hectares { get; }
    public string CropType { get; }
    public string SoilType { get; }

    // Defaults conservadores até que o agrônomo configure o SLA real
    // via UpdateSlaLimits (evento ZoneSlaLimitsUpdated).
    public double MinSoilMoisturePercent { get; private set; } = 0;
    public double MaxSoilMoisturePercent { get; private set; } = 100;

    public IReadOnlyCollection<IDomainEvent> DomainEvents => _domainEvents.AsReadOnly();

    private Zone(string zoneId, string name, decimal hectares, string cropType, string soilType)
    {
        ZoneId = zoneId;
        Name = name;
        Hectares = hectares;
        CropType = cropType;
        SoilType = soilType;
    }

    public static Zone Register(string zoneId, string name, decimal hectares, string cropType, string soilType)
    {
        if (string.IsNullOrWhiteSpace(zoneId))
            throw new DomainValidationException("ZoneId is required.");

        if (hectares <= 0)
            throw new DomainValidationException("Hectares must be greater than zero.");

        var zone = new Zone(zoneId, name, hectares, cropType, soilType);

        zone._domainEvents.Add(new ZoneRegistered(
            EventId: Guid.NewGuid(),
            OccurredAt: DateTime.UtcNow,
            ZoneId: zoneId,
            Name: name,
            Hectares: hectares,
            CropType: cropType,
            SoilType: soilType));

        return zone;
    }

    public void UpdateSlaLimits(double minSoilMoisturePercent, double maxSoilMoisturePercent)
    {
        if (minSoilMoisturePercent < 0 || maxSoilMoisturePercent > 100)
            throw new DomainValidationException("SLA limits must be within the 0-100 range.");

        if (minSoilMoisturePercent >= maxSoilMoisturePercent)
            throw new DomainValidationException("Minimum SLA limit must be lower than the maximum.");

        MinSoilMoisturePercent = minSoilMoisturePercent;
        MaxSoilMoisturePercent = maxSoilMoisturePercent;

        _domainEvents.Add(new ZoneSlaLimitsUpdated(
            EventId: Guid.NewGuid(),
            OccurredAt: DateTime.UtcNow,
            ZoneId: ZoneId,
            MinSoilMoisturePercent: minSoilMoisturePercent,
            MaxSoilMoisturePercent: maxSoilMoisturePercent));
    }

    /// <summary>
    /// Called by the application layer after domain events have been
    /// dispatched (e.g. published to Kafka), so the same events aren't
    /// re-published if this aggregate instance is reused.
    /// </summary>
    public void ClearDomainEvents() => _domainEvents.Clear();
}
