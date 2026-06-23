# 🐹 telemetry-ingestion (Go)

**Generic Subdomain** do DropSense — o "porteiro da infraestrutura". Ver `docs/01-CONTEXTO-PROJETO.md` na raiz do repositório para o papel deste serviço na arquitetura geral.

## Status atual

- [x] Validação física na porta (`SoilReading.Validate()`)
- [x] Handler HTTP (`POST /readings`) — decide 202/400 de forma síncrona
- [x] Pipeline com canal + worker pool — publica de forma assíncrona/concorrente
- [x] `Publisher` stub (`StdoutPublisher`) — log no lugar do Kafka real
- [x] Dockerfile multi-stage
- [ ] Producer Kafka de verdade (`internal/kafka`), publicando `SoilReadingRegistered` em `telemetry.readings.v1`
- [ ] Worker periódico consultando a API externa de clima → `WeatherForecastUpdated`
- [ ] Consumer de `zone.events.v1` (cache local de zonas válidas)
- [ ] Métricas Prometheus (descartes, latência, throughput)

## Decisão de design: validação síncrona, publish assíncrono

A regra definida pra esse serviço foi "descarta na porta tudo que fere a física". Isso é feito **de forma síncrona no handler HTTP** — não dentro do canal. O motivo:

- Validar `0% ≤ umidade ≤ 100%` é checagem de CPU pura, sem I/O — não é isso que justifica um canal.
- O canal existe pra proteger contra a parte **lenta**: o publish no Kafka (chamada de rede). Um publish lento não pode travar a próxima requisição HTTP entrando.

Por isso:

```
POST /readings
   │
   ▼
decodifica JSON → valida (síncrono) ──X──> 400 Bad Request (na hora)
   │
   ✓ válido
   │
   ▼
pipeline.Submit() → canal bufferizado → worker pool → Publisher.Publish()
   │
   ▼
202 Accepted (handler já respondeu, não espera o publish terminar)
```

## Estrutura

```
cmd/server/main.go              ← wiring: HTTP + pipeline + publisher
internal/
  ingestion/
    reading.go                  ← SoilReading + Validate()
    reading_test.go
    pipeline.go                 ← canal + worker pool
    pipeline_test.go
    publisher.go                ← interface Publisher + StdoutPublisher (stub)
  httpapi/
    handler.go                  ← decodifica, valida, decide 202/400
    handler_test.go
Dockerfile                      ← multi-stage: golang:1.22-alpine (build) → alpine:3.19 (runtime)
```

## Por que `alpine` no runtime e não `scratch`?

`scratch` (imagem totalmente vazia) seria menor ainda, mas esse serviço vai fazer uma chamada HTTPS pra API de clima no próximo passo — e `scratch` não tem certificados CA, então a validação TLS falharia. `alpine:3.19` com `ca-certificates` resolve isso e ainda fica pequena (~5-8MB), além de dar um shell pra debug via `docker exec` se precisar.

## Como rodar

**Localmente (com Go instalado, ou via `docker run` usando a imagem oficial como CLI):**
```bash
go test ./... -v -race   # 13 testes, devem passar
go run ./cmd/server       # sobe em :8080
```

**Via Docker Compose (recomendado, já que você não tem Go instalado no host):**
```bash
cd ../../infra
docker compose up --build telemetry-ingestion
```

**Testando manualmente:**
```bash
# Leitura válida → 202
curl -X POST http://localhost:8080/readings \
  -H "Content-Type: application/json" \
  -d '{"sensorId":"sensor-04812","zoneId":"zone-042","soilMoisturePercent":38.5,"soilTemperatureCelsius":24.1,"measuredAt":"2026-06-23T14:30:00Z"}'

# Leitura impossível (umidade > 100%) → 400
curl -X POST http://localhost:8080/readings \
  -H "Content-Type: application/json" \
  -d '{"sensorId":"sensor-099","zoneId":"zone-042","soilMoisturePercent":150,"soilTemperatureCelsius":24.1,"measuredAt":"2026-06-23T14:30:00Z"}'
```

## Decisões registradas

- **Por que `Publisher` é uma interface?** Pra poder testar `Pipeline` sem Kafka rodando. O `StdoutPublisher` de hoje será substituído por um `KafkaPublisher` real sem mexer em `Pipeline` nem no handler HTTP — é a fronteira certa entre lógica e infraestrutura.
- **Por que `Submitter` (no `httpapi`) é uma interface menor que `Pipeline` inteira?** O handler só precisa de `Submit(reading)` — não precisa saber que existe canal, worker, contexto. Isso deixa o teste do handler livre de qualquer concorrência.
- **Por que os testes do `Pipeline` usam `waitUntil` (polling) em vez de `time.Sleep` fixo?** Sleep fixo é frágil (ou espera demais, ou às vezes falha em máquina lenta). Polling com timeout é o padrão correto pra testar código concorrente.
