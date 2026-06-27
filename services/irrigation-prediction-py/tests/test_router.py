import json
import logging
from datetime import timedelta

from app.events import SoilReadingPayload, WeatherForecastPayload, new_envelope
from app.router import handle_message
from app.window import WindowAggregator


def _raw_soil_reading_envelope() -> bytes:
    payload = SoilReadingPayload(
        sensor_id="sensor-04812",
        zone_id="zone-042",
        soil_moisture_percent=38.5,
        soil_temperature_celsius=24.1,
        measured_at="2026-06-23T14:30:00Z",
    )
    return json.dumps(new_envelope("SoilReadingRegistered", 1, "corr-1", payload)).encode("utf-8")


def _raw_weather_forecast_envelope() -> bytes:
    payload = WeatherForecastPayload(
        zone_id="zone-042",
        rain_probability_percent=80,
        forecast_temperature_celsius=29.5,
        forecast_window_hours=12,
        source="open-meteo",
    )
    return json.dumps(new_envelope("WeatherForecastUpdated", 1, "corr-2", payload)).encode("utf-8")


def test_handle_message_with_soil_reading_feeds_the_aggregator():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))

    handle_message(_raw_soil_reading_envelope(), aggregator)

    assert "zone-042" in aggregator.known_zones()


def test_handle_message_with_weather_forecast_feeds_the_aggregator():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    # precisa de uma leitura de solo para a zona aparecer numa janela -
    # o forecast por si só não cria uma zona "conhecida" (ver
    # test_window.py: known_zones só lista zonas com leitura).
    handle_message(_raw_soil_reading_envelope(), aggregator)
    handle_message(_raw_weather_forecast_envelope(), aggregator)

    from datetime import datetime, timezone

    window = aggregator.compute_window("zone-042", datetime.now(timezone.utc))

    assert window is not None
    assert window.rain_probability_percent == 80


def test_handle_message_with_unknown_event_type_does_not_raise(caplog):
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    raw = json.dumps(
        {
            "eventId": "x",
            "eventType": "SomeFutureEvent",
            "eventVersion": 1,
            "occurredAt": "2026-06-23T14:30:00Z",
            "producer": "someone-else",
            "correlationId": "corr-3",
            "payload": {},
        }
    ).encode("utf-8")

    with caplog.at_level(logging.WARNING):
        handle_message(raw, aggregator)  # não deve levantar exceção

    assert aggregator.known_zones() == []


def test_handle_message_with_malformed_json_does_not_raise(caplog):
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))

    with caplog.at_level(logging.ERROR):
        handle_message(b"{not valid json", aggregator)  # não deve levantar exceção

    assert aggregator.known_zones() == []
