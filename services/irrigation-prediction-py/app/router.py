"""Despacha uma mensagem crua recebida do Kafka para o WindowAggregator,
de acordo com o eventType do envelope.

Separado da pluggagem de Kafka (app/kafka_consumer.py) de propósito:
esta função é pura o suficiente para ser testada sem broker nenhum -
mesma fronteira que handler.go usa no Go (decodifica e decide o que
fazer, sem saber nada sobre rede).

Nunca levanta exceção: uma mensagem mal formada ou de um tipo
desconhecido é logada e descartada, não derruba a thread do
consumidor. Mesma filosofia de "loga e segue" do Pipeline.worker e do
Scheduler.tick no Go.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone

from app.events import SoilReadingPayload, WeatherForecastPayload, parse_envelope
from app.window import WindowAggregator

logger = logging.getLogger(__name__)

_SOIL_READING_EVENT_TYPE = "SoilReadingRegistered"
_WEATHER_FORECAST_EVENT_TYPE = "WeatherForecastUpdated"


def handle_message(raw: bytes, aggregator: WindowAggregator) -> None:
    try:
        envelope = parse_envelope(raw)
    except Exception:
        logger.exception("failed to parse envelope, discarding message")
        return

    try:
        if envelope.event_type == _SOIL_READING_EVENT_TYPE:
            payload = SoilReadingPayload.model_validate(envelope.payload)
            aggregator.add_soil_reading(payload, received_at=datetime.now(timezone.utc))
        elif envelope.event_type == _WEATHER_FORECAST_EVENT_TYPE:
            forecast = WeatherForecastPayload.model_validate(envelope.payload)
            aggregator.set_weather_forecast(forecast)
        else:
            logger.warning("unknown event type %s, ignoring", envelope.event_type)
    except Exception:
        logger.exception("failed to handle envelope of type %s, discarding message", envelope.event_type)
