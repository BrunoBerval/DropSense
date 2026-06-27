using System.Text.Json.Serialization;

namespace FarmManagement.Api.Events;

// Envelope<TPayload> é o formato padrão de evento do DropSense,
// conforme o README ("EVENT CONTRACTS - SOURCE OF TRUTH" > Event
// Envelope) - terceira implementação desse mesmo contrato no
// projeto, depois de events.Envelope (Go) e app/events.Envelope
// (Python). Mesma ideia de Anti-Corruption Layer das outras duas: o
// [JsonPropertyName] aqui faz o mesmo papel que json:"camelCase" faz
// nas structs em Go e Field(alias=...) faz nos modelos Pydantic.
//
// Ser genérico em TPayload (em vez de Payload como objeto bruto) é
// uma escolha possível em C# que não existia do mesmo jeito nas
// outras duas linguagens: como este serviço consome só UM tipo de
// evento (IrrigationDecisionCalculated, de um único tópico), não há
// necessidade de um passo de "decidir o tipo a partir do eventType
// antes de desserializar o payload" como o router.go/router.py
// precisam fazer (eles consomem dois tipos de evento, de dois
// tópicos diferentes).
public sealed class Envelope<TPayload>
{
    [JsonPropertyName("eventId")]
    public string EventId { get; init; } = "";

    [JsonPropertyName("eventType")]
    public string EventType { get; init; } = "";

    [JsonPropertyName("eventVersion")]
    public int EventVersion { get; init; }

    [JsonPropertyName("occurredAt")]
    public string OccurredAt { get; init; } = "";

    [JsonPropertyName("producer")]
    public string Producer { get; init; } = "";

    [JsonPropertyName("correlationId")]
    public string CorrelationId { get; init; } = "";

    [JsonPropertyName("payload")]
    public TPayload Payload { get; init; } = default!;

    // NewEnvelope (Go) / new_envelope (Python) - mesmo papel aqui.
    // eventId e occurredAt são gerados na borda de publicação, não
    // são decisão de negócio de quem chama. occurredAt usa o mesmo
    // formato RFC3339 sem fração de segundo que o resto do contrato
    // de fio do projeto já usa.
    public static Envelope<TPayload> Wrap(string eventType, string correlationId, TPayload payload) => new()
    {
        EventId = Guid.NewGuid().ToString(),
        EventType = eventType,
        EventVersion = EventTypes.CurrentVersion,
        OccurredAt = DateTime.UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ"),
        Producer = "farm-management",
        CorrelationId = correlationId,
        Payload = payload,
    };
}
