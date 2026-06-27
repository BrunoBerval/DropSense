using FarmManagement.Api.Application;
using FarmManagement.Api.Events;
using FarmManagement.Api.Infrastructure;

var builder = WebApplication.CreateBuilder(args);

// Mesmos defaults dos outros dois serviços: "kafka:9092" é o nome do
// serviço dentro da rede do docker-compose.
var kafkaBrokers = Environment.GetEnvironmentVariable("KAFKA_BROKERS") ?? "kafka:9092";
var consumerGroup = Environment.GetEnvironmentVariable("KAFKA_CONSUMER_GROUP") ?? "farm-management";

// PROVISÓRIO: seed estática em vez de um repositório real - ver o
// comentário em StaticZoneRepository.cs.
builder.Services.AddSingleton<IZoneRepository, StaticZoneRepository>();

builder.Services.AddSingleton<IRawProducer>(_ => new ConfluentRawProducer(kafkaBrokers));
builder.Services.AddSingleton<IIrrigationEventPublisher, KafkaIrrigationEventPublisher>();

builder.Services.AddSingleton<IRawConsumer>(
    _ => new ConfluentRawConsumer(kafkaBrokers, consumerGroup, Topics.IrrigationDecisions));

builder.Services.AddSingleton<DecisionHandler>();
builder.Services.AddHostedService<DecisionConsumerWorker>();

var app = builder.Build();

app.MapGet("/healthz", () => Results.Ok(new { status = "ok" }));

app.Run();
