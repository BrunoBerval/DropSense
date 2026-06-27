using FarmManagement.Api.Domain;

namespace FarmManagement.Tests;

public class ZoneTests
{
    [Fact]
    public void EvaluateIrrigationRequest_NotUnderMaintenance_SufficientReservoir_Approves()
    {
        var zone = new Zone("zone-042", isUnderMaintenance: false, new Reservoir(hasSufficientVolume: true));

        var evaluation = zone.EvaluateIrrigationRequest();

        Assert.True(evaluation.IsApproved);
        Assert.Null(evaluation.Reason);
    }

    [Fact]
    public void EvaluateIrrigationRequest_UnderMaintenance_Rejects_EvenWithSufficientReservoir()
    {
        var zone = new Zone("zone-042", isUnderMaintenance: true, new Reservoir(hasSufficientVolume: true));

        var evaluation = zone.EvaluateIrrigationRequest();

        Assert.False(evaluation.IsApproved);
        Assert.Equal(RejectionReason.ZoneUnderMaintenance, evaluation.Reason);
    }

    [Fact]
    public void EvaluateIrrigationRequest_InsufficientReservoir_Rejects()
    {
        var zone = new Zone("zone-042", isUnderMaintenance: false, new Reservoir(hasSufficientVolume: false));

        var evaluation = zone.EvaluateIrrigationRequest();

        Assert.False(evaluation.IsApproved);
        Assert.Equal(RejectionReason.ReservoirInsufficientVolume, evaluation.Reason);
    }

    [Fact]
    public void EvaluateIrrigationRequest_BothFail_MaintenanceTakesPriority()
    {
        // Mesma ordem do README: "A bomba da Zona X está em
        // manutenção? O reservatório de água tem volume suficiente?"
        // - manutenção é checada primeiro, então é o motivo
        // reportado mesmo quando os dois falham.
        var zone = new Zone("zone-042", isUnderMaintenance: true, new Reservoir(hasSufficientVolume: false));

        var evaluation = zone.EvaluateIrrigationRequest();

        Assert.Equal(RejectionReason.ZoneUnderMaintenance, evaluation.Reason);
    }
}
