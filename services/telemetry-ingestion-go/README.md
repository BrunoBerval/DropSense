# 🐹 telemetry-ingestion (Go)

Ainda não iniciado. Ver `docs/01-CONTEXTO-PROJETO.md` na raiz do repo para o papel deste serviço na arquitetura (Generic Subdomain - "porteiro da infraestrutura").

## Responsabilidades planejadas
- Receber leituras de sensor via HTTP, validar formato/limites físicos, publicar `SoilReadingRegistered` em `telemetry.readings.v1`.
- Consultar API externa de clima periodicamente, publicar `WeatherForecastUpdated` em `weather.forecasts.v1`.
- Consumir `zone.events.v1` para manter cache local de zonas válidas (e rejeitar leitura de zona desconhecida/desativada).
