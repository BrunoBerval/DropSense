from datetime import timedelta

from app.config import get_settings


def test_get_settings_uses_defaults_when_no_env_vars_set(monkeypatch):
    monkeypatch.delenv("KAFKA_BROKERS", raising=False)
    monkeypatch.delenv("WINDOW_SIZE_SECONDS", raising=False)
    monkeypatch.delenv("DECISION_INTERVAL_SECONDS", raising=False)

    settings = get_settings()

    assert settings.kafka_brokers == "kafka:9092"
    assert settings.window_size == timedelta(minutes=5)
    assert settings.decision_interval == timedelta(minutes=1)


def test_get_settings_respects_overrides(monkeypatch):
    monkeypatch.setenv("KAFKA_BROKERS", "broker1:9092,broker2:9092")
    monkeypatch.setenv("WINDOW_SIZE_SECONDS", "120")

    settings = get_settings()

    assert settings.kafka_brokers == "broker1:9092,broker2:9092"
    assert settings.window_size == timedelta(seconds=120)


def test_get_settings_falls_back_on_invalid_duration(monkeypatch):
    monkeypatch.setenv("WINDOW_SIZE_SECONDS", "not-a-number")

    settings = get_settings()

    assert settings.window_size == timedelta(minutes=5)
