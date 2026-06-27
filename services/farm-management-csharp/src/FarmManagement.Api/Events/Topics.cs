namespace FarmManagement.Api.Events;

// Tópicos conforme a tabela de tópicos do README.
public static class Topics
{
    public const string IrrigationDecisions = "irrigation.decisions.v1";
    public const string IrrigationEvents = "irrigation.events.v1";
}

// Tipos e versões de evento conforme "EVENT CONTRACTS - SOURCE OF
// TRUTH" no README. IrrigationStarted e IrrigationRejected
// compartilham o tópico irrigation.events.v1 (ver Topics acima), mas
// cada um tem seu próprio eventType.
public static class EventTypes
{
    public const string IrrigationDecisionCalculated = "IrrigationDecisionCalculated";
    public const string IrrigationStarted = "IrrigationStarted";
    public const string IrrigationRejected = "IrrigationRejected";

    public const int CurrentVersion = 1;
}

// Decision conforme o "Decision Enum" do IrrigationDecisionCalculated
// no README - strings, não um enum C# aqui, porque são valores que
// chegam de fora (publicados pelo Python) e precisam bater
// exatamente com o que ele manda, sem depender de uma conversão.
public static class Decisions
{
    public const string StartIrrigation = "START_IRRIGATION";
    public const string SkipIrrigation = "SKIP_IRRIGATION";
}
