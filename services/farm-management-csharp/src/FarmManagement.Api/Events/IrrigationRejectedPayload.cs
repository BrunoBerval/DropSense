using System.Text.Json.Serialization;
using FarmManagement.Api.Domain;

namespace FarmManagement.Api.Events;

// Payload de IrrigationRejected, conforme o README ("EVENT CONTRACTS -
// SOURCE OF TRUTH" > IrrigationRejected > Payload + Reason Enum).
public sealed class IrrigationRejectedPayload
{
    [JsonPropertyName("zoneId")]
    public string ZoneId { get; init; } = "";

    [JsonPropertyName("reason")]
    public string Reason { get; init; } = "";

    [JsonPropertyName("rejectedAt")]
    public string RejectedAt { get; init; } = "";
}

// O enum RejectionReason (Domain) é PascalCase, idiomático em C#; o
// contrato publicado usa SCREAMING_SNAKE_CASE (mesmas strings que o
// README documenta: ZONE_UNDER_MAINTENANCE,
// RESERVOIR_INSUFFICIENT_VOLUME). Essa extensão é a tradução
// explícita entre os dois - sem ela, um .ToString() ingênuo no enum
// publicaria "ZoneUnderMaintenance" e quebraria o contrato
// silenciosamente, sem erro de compilação nenhum.
public static class RejectionReasonExtensions
{
    public static string ToContractString(this RejectionReason reason) => reason switch
    {
        RejectionReason.ZoneUnderMaintenance => "ZONE_UNDER_MAINTENANCE",
        RejectionReason.ReservoirInsufficientVolume => "RESERVOIR_INSUFFICIENT_VOLUME",
        _ => throw new ArgumentOutOfRangeException(nameof(reason), reason, "unmapped rejection reason"),
    };
}
