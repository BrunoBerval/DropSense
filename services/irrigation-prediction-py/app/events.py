"""Contrato de eventos do DropSense, do lado do consumidor/produtor em
Python.

Espelha deliberadamente o pacote internal/events do serviço Go: o
Envelope genérico (eventId, eventType, eventVersion, occurredAt,
producer, correlationId, payload) definido no README
("EVENT CONTRACTS - SOURCE OF TRUTH" > Event Envelope), e uma classe
de payload por tipo de evento - cada uma com os nomes JSON exatos do
contrato publicado, via Field(alias=...) do Pydantic. É o mesmo papel
que `json:"camelCase"` faz nas structs em Go: a fronteira entre o
nome idiomático da linguagem (snake_case aqui, PascalCase lá) e o
nome publicado no contrato (sempre camelCase).
"""

from __future__ import annotations

import json
import uuid
from datetime import datetime, timezone

from pydantic import BaseModel, ConfigDict, Field

PRODUCER_NAME = "irrigation-prediction"

# Tópicos conforme a tabela de tópicos do README.
SOIL_READING_TOPIC = "telemetry.readings.v1"
WEATHER_FORECAST_TOPIC = "weather.forecasts.v1"
IRRIGATION_DECISION_TOPIC = "irrigation.decisions.v1"

IRRIGATION_DECISION_EVENT_TYPE = "IrrigationDecisionCalculated"
IRRIGATION_DECISION_EVENT_VERSION = 1


class _ContractModel(BaseModel):
    """Base comum: aceita popular tanto pelo nome do campo Python
    (snake_case, conveniente para construir programaticamente) quanto
    pelo alias JSON (camelCase, necessário para fazer parse de
    mensagens recebidas do Kafka)."""

    model_config = ConfigDict(populate_by_name=True)


class SoilReadingPayload(_ContractModel):
    """Payload de SoilReadingRegistered - mesmos campos e nomes JSON
    de events.SoilReadingPayload no Go."""

    sensor_id: str = Field(alias="sensorId")
    zone_id: str = Field(alias="zoneId")
    soil_moisture_percent: float = Field(alias="soilMoisturePercent")
    soil_temperature_celsius: float = Field(alias="soilTemperatureCelsius")
    measured_at: str = Field(alias="measuredAt")


class WeatherForecastPayload(_ContractModel):
    """Payload de WeatherForecastUpdated - mesmos campos e nomes JSON
    de events.WeatherForecastPayload no Go."""

    zone_id: str = Field(alias="zoneId")
    rain_probability_percent: int = Field(alias="rainProbabilityPercent")
    forecast_temperature_celsius: float = Field(alias="forecastTemperatureCelsius")
    forecast_window_hours: int = Field(alias="forecastWindowHours")
    source: str


class IrrigationDecisionPayload(_ContractModel):
    """Payload de IrrigationDecisionCalculated - este é o evento que
    ESTE serviço produz, conforme o README
    ("EVENT CONTRACTS - SOURCE OF TRUTH" > Irrigation Prediction
    Events). Ordem dos campos importa para o teste de contrato byte a
    byte: segue a mesma ordem do exemplo no README."""

    zone_id: str = Field(alias="zoneId")
    decision: str
    window_start: str = Field(alias="windowStart")
    window_end: str = Field(alias="windowEnd")
    average_soil_moisture_percent: float = Field(alias="averageSoilMoisturePercent")
    rain_probability_percent: int = Field(alias="rainProbabilityPercent")
    confidence_score: float = Field(alias="confidenceScore")
    model_version: str = Field(alias="modelVersion")


class Envelope(_ContractModel):
    """Envelope genérico recebido - payload fica como dict bruto
    (ainda não traduzido para um payload específico); quem chama
    decide qual *Payload usar, a partir de event_type."""

    event_id: str = Field(alias="eventId")
    event_type: str = Field(alias="eventType")
    event_version: int = Field(alias="eventVersion")
    occurred_at: str = Field(alias="occurredAt")
    producer: str
    correlation_id: str = Field(alias="correlationId")
    payload: dict


def new_envelope(event_type: str, event_version: int, correlation_id: str, payload: BaseModel) -> dict:
    """Monta o envelope em volta de um payload já no formato de fio
    (ex.: IrrigationDecisionPayload). Espelha events.NewEnvelope no Go:
    eventId e occurredAt são gerados aqui, na borda de publicação -
    não são decisão de negócio de quem chama. occurredAt usa o mesmo
    formato (RFC3339, sem fração de segundo) que o resto do contrato
    de fio do projeto já usa, batendo com o exemplo do README
    ("2026-06-23T14:32:10Z").
    """
    return {
        "eventId": str(uuid.uuid4()),
        "eventType": event_type,
        "eventVersion": event_version,
        "occurredAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "producer": PRODUCER_NAME,
        "correlationId": correlation_id,
        "payload": payload.model_dump(by_alias=True),
    }


def parse_envelope(raw: bytes) -> Envelope:
    """Faz parse de uma mensagem crua recebida do Kafka para um
    Envelope. payload fica como dict bruto - quem chama traduz para
    SoilReadingPayload ou WeatherForecastPayload de acordo com
    event_type (ver app/router.py)."""
    return Envelope.model_validate(json.loads(raw))
