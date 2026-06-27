namespace FarmManagement.Api.Domain;

// Resultado de Zone.EvaluateIrrigationRequest() - aprovado, ou
// rejeitado com um motivo específico. Em vez de um bool solto +
// um motivo opcional separado, agrupar os dois aqui torna impossível
// representar o estado inválido "aprovado, mas com motivo de rejeição
// preenchido" - o construtor privado garante que só os dois estados
// válidos (Approved ou Rejected-com-motivo) podem existir.
public sealed class IrrigationEvaluation
{
    public bool IsApproved { get; }
    public RejectionReason? Reason { get; }

    private IrrigationEvaluation(bool isApproved, RejectionReason? reason)
    {
        IsApproved = isApproved;
        Reason = reason;
    }

    public static IrrigationEvaluation Approved() => new(true, null);

    public static IrrigationEvaluation Rejected(RejectionReason reason) => new(false, reason);
}
