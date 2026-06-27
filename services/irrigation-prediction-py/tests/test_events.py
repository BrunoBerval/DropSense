import json
import uuid

from app.events import (
    IRRIGATION_DECISION_EVENT_TYPE,
    IRRIGATION_DECISION_EVENT_VERSION,
    IRRIGATION_DECISION_TOPIC,
    SOIL_READING_TOPIC,
    WEATHER_FORECAST_TOPIC,
    IrrigationDecisionPayload,
    SoilReadingPayload,
    WeatherForecastPayload,
    new_envelope,
    parse_envelope,
)


def test_topic_constants_match_readme_table():
    # Tabela de tópicos do README - travado aqui para que qualquer
    # divergência futura quebre um teste, não um deploy silencioso.
    assert SOIL_READING_TOPIC == "telemetry.readings.v1"
    assert WEATHER_FORECAST_TOPIC == "weather.forecasts.v1"
    assert IRRIGATION_DECISION_TOPIC == "irrigation.decisions.v1"


def test_soil_reading_payload_parses_readme_example():
    # Exemplo literal do README, seção SoilReadingRegistered > Payload.
    raw = {
        "sensorId": "sensor-04812",
        "zoneId": "zone-042",
        "soilMoisturePercent": 38.5,
        "soilTemperatureCelsius": 24.1,
        "measuredAt": "2026-06-23T14:30:00Z",
    }

    payload = SoilReadingPayload.model_validate(raw)

    assert payload.sensor_id == "sensor-04812"
    assert payload.zone_id == "zone-042"
    assert payload.soil_moisture_percent == 38.5
    assert payload.soil_temperature_celsius == 24.1
    assert payload.measured_at == "2026-06-23T14:30:00Z"


def test_weather_forecast_payload_parses_readme_example():
    # Exemplo literal do README, seção WeatherForecastUpdated > Payload.
    raw = {
        "zoneId": "zone-042",
        "rainProbabilityPercent": 80,
        "forecastTemperatureCelsius": 29.5,
        "forecastWindowHours": 12,
        "source": "open-meteo",
    }

    payload = WeatherForecastPayload.model_validate(raw)

    assert payload.zone_id == "zone-042"
    assert payload.rain_probability_percent == 80
    assert payload.forecast_temperature_celsius == 29.5
    assert payload.forecast_window_hours == 12
    assert payload.source == "open-meteo"


def test_irrigation_decision_payload_json_matches_readme_contract():
    # Mesmo papel do teste Go TestSoilReadingPayload_JSON_MatchesReadmeContract:
    # compara byte a byte com o exemplo do README, seção
    # IrrigationDecisionCalculated > Payload.
    payload = IrrigationDecisionPayload(
        zone_id="zone-042",
        decision="START_IRRIGATION",
        window_start="2026-06-23T14:25:00Z",
        window_end="2026-06-23T14:30:00Z",
        average_soil_moisture_percent=31.2,
        rain_probability_percent=80,
        confidence_score=0.74,
        model_version="v1",
    )

    got = json.dumps(payload.model_dump(by_alias=True))
    want = json.dumps(
        {
            "zoneId": "zone-042",
            "decision": "START_IRRIGATION",
            "windowStart": "2026-06-23T14:25:00Z",
            "windowEnd": "2026-06-23T14:30:00Z",
            "averageSoilMoisturePercent": 31.2,
            "rainProbabilityPercent": 80,
            "confidenceScore": 0.74,
            "modelVersion": "v1",
        }
    )

    assert got == want


def test_new_envelope_sets_caller_provided_fields():
    payload = WeatherForecastPayload(
        zone_id="zone-042",
        rain_probability_percent=80,
        forecast_temperature_celsius=29.5,
        forecast_window_hours=12,
        source="open-meteo",
    )

    envelope = new_envelope("WeatherForecastUpdated", 1, "corr-123", payload)

    assert envelope["eventType"] == "WeatherForecastUpdated"
    assert envelope["eventVersion"] == 1
    assert envelope["correlationId"] == "corr-123"
    assert envelope["payload"]["zoneId"] == "zone-042"


def test_new_envelope_generates_valid_uuid_event_id():
    payload = WeatherForecastPayload(
        zone_id="zone-042",
        rain_probability_percent=0,
        forecast_temperature_celsius=20.0,
        forecast_window_hours=12,
        source="open-meteo",
    )

    envelope = new_envelope("WeatherForecastUpdated", 1, "corr-123", payload)

    uuid.UUID(envelope["eventId"])  # não levanta exceção se for um UUID válido


def test_new_envelope_uses_contract_field_names():
    payload = WeatherForecastPayload(
        zone_id="zone-042",
        rain_probability_percent=0,
        forecast_temperature_celsius=20.0,
        forecast_window_hours=12,
        source="open-meteo",
    )

    envelope = new_envelope("WeatherForecastUpdated", 1, "corr-123", payload)

    for field in ("eventId", "eventType", "eventVersion", "occurredAt", "producer", "correlationId", "payload"):
        assert field in envelope


def test_parse_envelope_roundtrips_a_published_message():
    payload = SoilReadingPayload(
        sensor_id="sensor-04812",
        zone_id="zone-042",
        soil_moisture_percent=38.5,
        soil_temperature_celsius=24.1,
        measured_at="2026-06-23T14:30:00Z",
    )
    envelope_dict = new_envelope("SoilReadingRegistered", 1, "corr-abc", payload)
    raw = json.dumps(envelope_dict).encode("utf-8")

    parsed = parse_envelope(raw)

    assert parsed.event_type == "SoilReadingRegistered"
    assert parsed.correlation_id == "corr-abc"
    assert parsed.payload["sensorId"] == "sensor-04812"


def test_irrigation_decision_event_constants():
    assert IRRIGATION_DECISION_EVENT_TYPE == "IrrigationDecisionCalculated"
    assert IRRIGATION_DECISION_EVENT_VERSION == 1
