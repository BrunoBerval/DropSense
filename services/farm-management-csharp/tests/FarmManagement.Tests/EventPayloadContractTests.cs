using System.Text.Json;
using FarmManagement.Api.Domain;
using FarmManagement.Api.Events;

namespace FarmManagement.Tests;

// Mesmo papel de TestSoilReadingPayload_JSON_MatchesReadmeContract
// (Go) e test_irrigation_decision_payload_json_matches_readme_contract
// (Python): trava o contrato publicado contra os exemplos literais do
// README, seção "EVENT CONTRACTS - SOURCE OF TRUTH".
public class EventPayloadContractTests
{
    [Fact]
    public void IrrigationStartedPayload_Json_MatchesReadmeContract()
    {
        var payload = new IrrigationStartedPayload
        {
            ZoneId = "zone-042",
            CycleId = "9f8e7d6c-...",
            StartedAt = "2026-06-23T14:31:05Z",
        };

        var json = JsonSerializer.Serialize(payload);

        Assert.Equal(
            """{"zoneId":"zone-042","cycleId":"9f8e7d6c-...","startedAt":"2026-06-23T14:31:05Z"}""",
            json);
    }

    [Theory]
    [InlineData(RejectionReason.ZoneUnderMaintenance, "ZONE_UNDER_MAINTENANCE")]
    [InlineData(RejectionReason.ReservoirInsufficientVolume, "RESERVOIR_INSUFFICIENT_VOLUME")]
    public void RejectionReason_MapsToContractString(RejectionReason reason, string expected)
    {
        Assert.Equal(expected, reason.ToContractString());
    }

    [Fact]
    public void IrrigationRejectedPayload_Json_MatchesReadmeContract()
    {
        var payload = new IrrigationRejectedPayload
        {
            ZoneId = "zone-042",
            Reason = RejectionReason.ReservoirInsufficientVolume.ToContractString(),
            RejectedAt = "2026-06-23T14:31:05Z",
        };

        var json = JsonSerializer.Serialize(payload);

        Assert.Equal(
            """{"zoneId":"zone-042","reason":"RESERVOIR_INSUFFICIENT_VOLUME","rejectedAt":"2026-06-23T14:31:05Z"}""",
            json);
    }

    [Fact]
    public void IrrigationDecisionPayload_ParsesReadmeExample()
    {
        const string raw = """
            {
              "zoneId": "zone-042",
              "decision": "START_IRRIGATION",
              "windowStart": "2026-06-23T14:25:00Z",
              "windowEnd": "2026-06-23T14:30:00Z",
              "averageSoilMoisturePercent": 31.2,
              "rainProbabilityPercent": 80,
              "confidenceScore": 0.74,
              "modelVersion": "v1"
            }
            """;

        var payload = JsonSerializer.Deserialize<IrrigationDecisionPayload>(raw)!;

        Assert.Equal("zone-042", payload.ZoneId);
        Assert.Equal("START_IRRIGATION", payload.Decision);
        Assert.Equal(31.2, payload.AverageSoilMoisturePercent);
        Assert.Equal(80, payload.RainProbabilityPercent);
        Assert.Equal(0.74, payload.ConfidenceScore);
        Assert.Equal("v1", payload.ModelVersion);
    }
}
