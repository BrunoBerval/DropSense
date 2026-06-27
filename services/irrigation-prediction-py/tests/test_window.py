from datetime import datetime, timedelta, timezone

from app.events import SoilReadingPayload, WeatherForecastPayload
from app.window import WindowAggregator


def make_reading(zone_id: str, moisture: float) -> SoilReadingPayload:
    return SoilReadingPayload(
        sensor_id="sensor-04812",
        zone_id=zone_id,
        soil_moisture_percent=moisture,
        soil_temperature_celsius=24.1,
        measured_at="2026-06-23T14:30:00Z",
    )


def make_forecast(zone_id: str, rain_probability: int) -> WeatherForecastPayload:
    return WeatherForecastPayload(
        zone_id=zone_id,
        rain_probability_percent=rain_probability,
        forecast_temperature_celsius=29.5,
        forecast_window_hours=12,
        source="open-meteo",
    )


def test_compute_window_returns_none_for_unknown_zone():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)

    assert aggregator.compute_window("zone-042", now) is None


def test_compute_window_averages_readings_within_window():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)

    aggregator.add_soil_reading(make_reading("zone-042", 30.0), received_at=now - timedelta(minutes=4))
    aggregator.add_soil_reading(make_reading("zone-042", 40.0), received_at=now - timedelta(minutes=1))

    window = aggregator.compute_window("zone-042", now)

    assert window is not None
    assert window.average_soil_moisture_percent == 35.0
    assert window.zone_id == "zone-042"
    assert window.window_end == now
    assert window.window_start == now - timedelta(minutes=5)


def test_compute_window_evicts_readings_outside_window():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)

    # fora da janela de 5min - não deve entrar na média.
    aggregator.add_soil_reading(make_reading("zone-042", 90.0), received_at=now - timedelta(minutes=10))
    aggregator.add_soil_reading(make_reading("zone-042", 30.0), received_at=now - timedelta(minutes=1))

    window = aggregator.compute_window("zone-042", now)

    assert window is not None
    assert window.average_soil_moisture_percent == 30.0


def test_compute_window_uses_latest_forecast_for_rain_probability():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)

    aggregator.add_soil_reading(make_reading("zone-042", 30.0), received_at=now)
    aggregator.set_weather_forecast(make_forecast("zone-042", 80))

    window = aggregator.compute_window("zone-042", now)

    assert window is not None
    assert window.rain_probability_percent == 80


def test_compute_window_defaults_rain_probability_to_zero_when_no_forecast_known():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)

    aggregator.add_soil_reading(make_reading("zone-042", 30.0), received_at=now)

    window = aggregator.compute_window("zone-042", now)

    assert window is not None
    assert window.rain_probability_percent == 0


def test_set_weather_forecast_overwrites_previous_value_for_same_zone():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)
    aggregator.add_soil_reading(make_reading("zone-042", 30.0), received_at=now)

    aggregator.set_weather_forecast(make_forecast("zone-042", 10))
    aggregator.set_weather_forecast(make_forecast("zone-042", 90))

    window = aggregator.compute_window("zone-042", now)

    assert window is not None
    assert window.rain_probability_percent == 90


def test_known_zones_lists_only_zones_with_readings():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)

    aggregator.add_soil_reading(make_reading("zone-042", 30.0), received_at=now)
    # forecast sem leitura nenhuma para a zona - não deveria aparecer.
    aggregator.set_weather_forecast(make_forecast("zone-099", 50))

    assert aggregator.known_zones() == ["zone-042"]


def test_zones_are_independent_of_each_other():
    aggregator = WindowAggregator(window_size=timedelta(minutes=5))
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)

    aggregator.add_soil_reading(make_reading("zone-042", 10.0), received_at=now)
    aggregator.add_soil_reading(make_reading("zone-099", 90.0), received_at=now)

    window_a = aggregator.compute_window("zone-042", now)
    window_b = aggregator.compute_window("zone-099", now)

    assert window_a.average_soil_moisture_percent == 10.0
    assert window_b.average_soil_moisture_percent == 90.0
