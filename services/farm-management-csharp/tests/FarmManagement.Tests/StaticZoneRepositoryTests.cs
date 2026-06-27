using FarmManagement.Api.Infrastructure;

namespace FarmManagement.Tests;

public class StaticZoneRepositoryTests
{
    [Fact]
    public void Find_SeededZone_ReturnsZoneNotUnderMaintenanceWithSufficientReservoir()
    {
        var repo = new StaticZoneRepository();

        var zone = repo.Find("zone-042");

        Assert.NotNull(zone);
        Assert.False(zone!.IsUnderMaintenance);
        Assert.True(zone.Reservoir.HasSufficientVolume);
    }

    [Fact]
    public void Find_UnknownZone_ReturnsNull()
    {
        var repo = new StaticZoneRepository();

        Assert.Null(repo.Find("zone-999"));
    }
}
