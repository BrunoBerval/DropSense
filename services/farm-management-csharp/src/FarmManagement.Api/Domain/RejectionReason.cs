namespace FarmManagement.Api.Domain;

// Mesmo enum de "Reason" do IrrigationRejected, conforme o README
// ("EVENT CONTRACTS - SOURCE OF TRUTH" > IrrigationRejected > Reason
// Enum). Os nomes aqui precisam bater exatamente com as strings que
// vão pro JSON publicado (ver Events/IrrigationRejectedPayload.cs).
public enum RejectionReason
{
    ZoneUnderMaintenance,
    ReservoirInsufficientVolume,
}
