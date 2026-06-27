using FarmManagement.Api.Events;

namespace FarmManagement.Api.Application;

// DecisionHandler é o que decide o que fazer com um
// IrrigationDecisionCalculated recebido - o equivalente, em forma, ao
// handle_message do Python (router.py) e ao corpo de
// ServeHTTP/Publish no Go: pega algo já decodificado e decide a
// próxima ação, sem saber nada sobre transporte (Kafka, HTTP, etc).
public sealed class DecisionHandler
{
    private readonly IZoneRepository _zones;
    private readonly IIrrigationEventPublisher _publisher;

    public DecisionHandler(IZoneRepository zones, IIrrigationEventPublisher publisher)
    {
        _zones = zones;
        _publisher = publisher;
    }

    public async Task HandleAsync(Envelope<IrrigationDecisionPayload> envelope)
    {
        // SKIP_IRRIGATION já é uma decisão completa vinda do Python -
        // não há nada para o Core Domain validar ou rejeitar aqui.
        // Só START_IRRIGATION passa pelas regras de negócio deste
        // serviço.
        if (envelope.Payload.Decision != Decisions.StartIrrigation)
        {
            return;
        }

        var zone = _zones.Find(envelope.Payload.ZoneId);
        if (zone is null)
        {
            // Zona desconhecida - loga e descarta (mesma filosofia de
            // "falha na borda, sem derrubar o consumidor" do resto do
            // projeto). Quem chama (o worker do Kafka) decide como logar.
            return;
        }

        var evaluation = zone.EvaluateIrrigationRequest();

        if (evaluation.IsApproved)
        {
            await _publisher.PublishIrrigationStartedAsync(zone.Id, envelope.CorrelationId);
        }
        else
        {
            await _publisher.PublishIrrigationRejectedAsync(zone.Id, evaluation.Reason!.Value, envelope.CorrelationId);
        }
    }
}
