using FarmManagement.Api.Domain;

namespace FarmManagement.Api.Application;

// Mesmo papel de ForecastFetcher/ForecastPublisher no weather.Scheduler
// do Go: quem CONSOME a interface (DecisionHandler) é quem a declara,
// não quem a implementa - assim a camada de aplicação não sabe nada
// sobre Kafka nem sobre como as zonas são armazenadas.
public interface IZoneRepository
{
    Zone? Find(string zoneId);
}

public interface IIrrigationEventPublisher
{
    Task PublishIrrigationStartedAsync(string zoneId, string correlationId);

    Task PublishIrrigationRejectedAsync(string zoneId, RejectionReason reason, string correlationId);
}
