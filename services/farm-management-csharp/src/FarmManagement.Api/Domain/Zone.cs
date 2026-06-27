namespace FarmManagement.Api.Domain;

// Zone é o agregado raiz deste Core Domain. A ordem das duas
// checagens em EvaluateIrrigationRequest segue exatamente a ordem em
// que o README as lista ("A bomba da Zona X está em manutenção? O
// reservatório de água tem volume suficiente?") - manutenção é
// checada primeiro.
//
// IsUnderMaintenance é mutável de propósito: por decisão do usuário,
// esse valor "deve ser informado pelo frontend" - hoje não existe
// frontend, então o valor fica fixo no que a seed estática definir
// (ver Infrastructure/StaticZoneRepository.cs), mas a propriedade já
// nasce pronta para ser atualizada quando essa fonte real existir,
// sem precisar mudar a forma da classe.
public sealed class Zone
{
    public string Id { get; }
    public bool IsUnderMaintenance { get; set; }
    public Reservoir Reservoir { get; }

    public Zone(string id, bool isUnderMaintenance, Reservoir reservoir)
    {
        Id = id;
        IsUnderMaintenance = isUnderMaintenance;
        Reservoir = reservoir;
    }

    public IrrigationEvaluation EvaluateIrrigationRequest()
    {
        if (IsUnderMaintenance)
        {
            return IrrigationEvaluation.Rejected(RejectionReason.ZoneUnderMaintenance);
        }

        if (!Reservoir.HasSufficientVolume)
        {
            return IrrigationEvaluation.Rejected(RejectionReason.ReservoirInsufficientVolume);
        }

        return IrrigationEvaluation.Approved();
    }
}
