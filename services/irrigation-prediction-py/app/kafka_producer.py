"""Publica IrrigationDecisionCalculated no Kafka.

O client real (confluent_kafka.Producer) é injetável via parâmetro
`client` exatamente pelo motivo de produceClient no Go: permite
testar a montagem da mensagem (tópico, chave de partição, envelope)
sem precisar de um broker real durante "pytest".
"""

from __future__ import annotations

import json
import uuid

from app.decision import MODEL_VERSION, Decision
from app.events import (
    IRRIGATION_DECISION_EVENT_TYPE,
    IRRIGATION_DECISION_EVENT_VERSION,
    IRRIGATION_DECISION_TOPIC,
    IrrigationDecisionPayload,
    new_envelope,
)
from app.window import WindowResult


def _format_rfc3339(dt) -> str:
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


class DecisionPublisher:
    def __init__(self, brokers: str | None = None, client=None):
        if client is not None:
            self._client = client
        else:
            from confluent_kafka import Producer

            self._client = Producer({"bootstrap.servers": brokers})

    def publish(self, window: WindowResult, decision: Decision) -> None:
        payload = IrrigationDecisionPayload(
            zone_id=window.zone_id,
            decision=decision.decision,
            window_start=_format_rfc3339(window.window_start),
            window_end=_format_rfc3339(window.window_end),
            average_soil_moisture_percent=window.average_soil_moisture_percent,
            rain_probability_percent=window.rain_probability_percent,
            confidence_score=decision.confidence_score,
            model_version=MODEL_VERSION,
        )

        envelope = new_envelope(
            IRRIGATION_DECISION_EVENT_TYPE,
            IRRIGATION_DECISION_EVENT_VERSION,
            correlation_id=str(uuid.uuid4()),
            payload=payload,
        )

        self._client.produce(
            IRRIGATION_DECISION_TOPIC,
            key=window.zone_id.encode("utf-8"),
            value=json.dumps(envelope).encode("utf-8"),
        )
        self._client.poll(0)

    def close(self) -> None:
        # flush() bloqueia até toda mensagem em buffer ser entregue
        # (ou o timeout expirar) - chamado uma vez, no shutdown
        # gracioso, mesmo papel de Producer.Close() no Go.
        flush = getattr(self._client, "flush", None)
        if flush is not None:
            flush(timeout=5)
