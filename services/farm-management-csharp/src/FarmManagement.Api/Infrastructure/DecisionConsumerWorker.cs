using System.Text.Json;
using FarmManagement.Api.Application;
using FarmManagement.Api.Events;

namespace FarmManagement.Api.Infrastructure;

// DecisionConsumerWorker é o BackgroundService que efetivamente
// consome irrigation.decisions.v1 - equivalente, em forma, ao
// Pipeline.worker (Go) e ao loop de TopicConsumer (Python): roda em
// segundo plano, processa até o host pedir para parar (aqui via
// CancellationToken, o mesmo papel de context.Context no Go e de
// threading.Event no Python), e isola toda a parte "burra" de
// transporte (poll, parse, despachar) longe da lógica de negócio,
// que já mora em DecisionHandler.
public sealed class DecisionConsumerWorker : BackgroundService
{
    private readonly IRawConsumer _consumer;
    private readonly DecisionHandler _handler;
    private readonly ILogger<DecisionConsumerWorker> _logger;

    public DecisionConsumerWorker(IRawConsumer consumer, DecisionHandler handler, ILogger<DecisionConsumerWorker> logger)
    {
        _consumer = consumer;
        _handler = handler;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        while (!stoppingToken.IsCancellationRequested)
        {
            string? raw;
            try
            {
                raw = _consumer.ConsumeValue(TimeSpan.FromSeconds(1));
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "consumer error on {Topic}", Topics.IrrigationDecisions);
                continue;
            }

            if (raw is null)
            {
                continue;
            }

            try
            {
                var envelope = JsonSerializer.Deserialize<Envelope<IrrigationDecisionPayload>>(raw);
                if (envelope is null)
                {
                    _logger.LogWarning("discarding message that parsed to null envelope");
                    continue;
                }

                await _handler.HandleAsync(envelope);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "failed to handle message, discarding");
            }
        }

        _consumer.Close();
    }
}
