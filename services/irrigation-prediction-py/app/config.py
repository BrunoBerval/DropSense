"""Configuração via variável de ambiente - mesmo padrão de getEnv/
getEnvDuration no main.go do Go, sem framework de settings extra
(pydantic-settings seria mais uma dependência para algo que algumas
funções pequenas já resolvem)."""

from __future__ import annotations

import os
from dataclasses import dataclass
from datetime import timedelta


def _get_env(key: str, fallback: str) -> str:
    return os.environ.get(key) or fallback


def _get_env_seconds(key: str, fallback_seconds: float) -> timedelta:
    value = os.environ.get(key)
    if not value:
        return timedelta(seconds=fallback_seconds)
    try:
        return timedelta(seconds=float(value))
    except ValueError:
        return timedelta(seconds=fallback_seconds)


@dataclass(frozen=True)
class Settings:
    kafka_brokers: str
    consumer_group: str
    window_size: timedelta
    decision_interval: timedelta


def get_settings() -> Settings:
    return Settings(
        kafka_brokers=_get_env("KAFKA_BROKERS", "kafka:9092"),
        consumer_group=_get_env("KAFKA_CONSUMER_GROUP", "irrigation-prediction"),
        # Janela do README: "média de umidade da Zona A nos últimos 5
        # minutos".
        window_size=_get_env_seconds("WINDOW_SIZE_SECONDS", 300),
        # Recalcula decisão a cada 1min por padrão - mais frequente
        # que o forecast (30min no Go), porque leituras de solo chegam
        # a cada 30s e merecem reação mais rápida.
        decision_interval=_get_env_seconds("DECISION_INTERVAL_SECONDS", 60),
    )
