using System.Text.Json.Serialization;

namespace FarmManagement.Api.Events;

// Payload de IrrigationStarted, conforme o README ("EVENT CONTRACTS -
// SOURCE OF TRUTH" > IrrigationStarted > Payload).
public sealed class IrrigationStartedPayload
{
    [JsonPropertyName("zoneId")]
    public string ZoneId { get; init; } = "";

    [JsonPropertyName("cycleId")]
    public string CycleId { get; init; } = "";

    [JsonPropertyName("startedAt")]
    public string StartedAt { get; init; } = "";
}
