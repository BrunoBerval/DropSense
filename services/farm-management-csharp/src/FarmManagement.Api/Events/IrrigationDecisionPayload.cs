using System.Text.Json.Serialization;

namespace FarmManagement.Api.Events;

// Payload de IrrigationDecisionCalculated - o evento que este serviço
// CONSOME, publicado pelo irrigation-prediction (Python). Mesmos
// campos e nomes JSON definidos no README ("EVENT CONTRACTS - SOURCE
// OF TRUTH" > IrrigationDecisionCalculated > Payload).
public sealed class IrrigationDecisionPayload
{
    [JsonPropertyName("zoneId")]
    public string ZoneId { get; init; } = "";

    [JsonPropertyName("decision")]
    public string Decision { get; init; } = "";

    [JsonPropertyName("windowStart")]
    public string WindowStart { get; init; } = "";

    [JsonPropertyName("windowEnd")]
    public string WindowEnd { get; init; } = "";

    [JsonPropertyName("averageSoilMoisturePercent")]
    public double AverageSoilMoisturePercent { get; init; }

    [JsonPropertyName("rainProbabilityPercent")]
    public int RainProbabilityPercent { get; init; }

    [JsonPropertyName("confidenceScore")]
    public double ConfidenceScore { get; init; }

    [JsonPropertyName("modelVersion")]
    public string ModelVersion { get; init; } = "";
}
