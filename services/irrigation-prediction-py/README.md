# 🐍 irrigation-prediction (Python)

Ainda não iniciado. Ver `docs/01-CONTEXTO-PROJETO.md` na raiz do repo para o papel deste serviço na arquitetura (Supporting Subdomain - "cérebro agronômico").

## Responsabilidades planejadas
- Consumir `telemetry.readings.v1` e `weather.forecasts.v1`.
- Agregar leituras por zona em tumbling window (5 min).
- Calcular `IrrigationDecisionCalculated` e publicar em `irrigation.decisions.v1`.
