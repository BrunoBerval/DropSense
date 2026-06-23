using FarmManagement.Domain.DomainEvents;
using FarmManagement.Domain.Exceptions;
using Xunit;

namespace FarmManagement.Domain.Tests;

public class ZoneTests
{
    // --- Invariante 1: registrar uma zona é o ponto de entrada do agregado ---

    [Fact]
    public void Register_WithValidData_ShouldCreateZoneAndRaiseZoneRegisteredEvent()
    {
        var zone = Zone.Register(
            zoneId: "zone-042",
            name: "Zona Norte - Lote 7",
            hectares: 42.5m,
            cropType: "COFFEE",
            soilType: "clay loam");

        Assert.Equal("zone-042", zone.ZoneId);
        Assert.Equal("Zona Norte - Lote 7", zone.Name);
        Assert.Equal(42.5m, zone.Hectares);

        // Toda mudança de estado relevante para o negócio precisa
        // resultar em um domain event - é isso que depois vira evento no Kafka.
        var raised = Assert.Single(zone.DomainEvents);
        var zoneRegistered = Assert.IsType<ZoneRegistered>(raised);
        Assert.Equal("zone-042", zoneRegistered.ZoneId);
        Assert.Equal("COFFEE", zoneRegistered.CropType);
    }

    [Fact]
    public void Register_WithoutZoneId_ShouldThrowDomainValidationException()
    {
        Assert.Throws<DomainValidationException>(() =>
            Zone.Register(zoneId: "", name: "Zona X", hectares: 10m, cropType: "COFFEE", soilType: "clay"));
    }

    [Theory]
    [InlineData(0)]
    [InlineData(-5)]
    public void Register_WithNonPositiveHectares_ShouldThrowDomainValidationException(decimal invalidHectares)
    {
        var exception = Assert.Throws<DomainValidationException>(() =>
            Zone.Register("zone-099", "Zona Inválida", invalidHectares, "COFFEE", "clay loam"));

        Assert.Contains("Hectares", exception.Message);
    }

    // --- Invariante 2: limites de SLA precisam fazer sentido (min < max, dentro de 0-100) ---

    [Fact]
    public void UpdateSlaLimits_WithValidLimits_ShouldUpdateZoneAndRaiseEvent()
    {
        var zone = Zone.Register("zone-042", "Zona Norte", 42.5m, "COFFEE", "clay loam");
        zone.ClearDomainEvents(); // ignora o ZoneRegistered, foco é só no evento desta ação

        zone.UpdateSlaLimits(minSoilMoisturePercent: 25, maxSoilMoisturePercent: 60);

        Assert.Equal(25, zone.MinSoilMoisturePercent);
        Assert.Equal(60, zone.MaxSoilMoisturePercent);

        var raised = Assert.Single(zone.DomainEvents);
        Assert.IsType<ZoneSlaLimitsUpdated>(raised);
    }

    [Fact]
    public void UpdateSlaLimits_WhenMinIsGreaterThanOrEqualToMax_ShouldThrowDomainValidationException()
    {
        var zone = Zone.Register("zone-042", "Zona Norte", 42.5m, "COFFEE", "clay loam");

        Assert.Throws<DomainValidationException>(() =>
            zone.UpdateSlaLimits(minSoilMoisturePercent: 60, maxSoilMoisturePercent: 25));
    }

    [Theory]
    [InlineData(-1, 50)]   // min abaixo de 0
    [InlineData(10, 101)]  // max acima de 100
    public void UpdateSlaLimits_OutsideZeroToHundredRange_ShouldThrowDomainValidationException(
        double min, double max)
    {
        var zone = Zone.Register("zone-042", "Zona Norte", 42.5m, "COFFEE", "clay loam");

        Assert.Throws<DomainValidationException>(() => zone.UpdateSlaLimits(min, max));
    }
}
