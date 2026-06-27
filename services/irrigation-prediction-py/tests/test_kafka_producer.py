import json
from datetime import datetime, timezone

from app.decision import Decision
from app.events import IRRIGATION_DECISION_TOPIC
from app.kafka_producer import DecisionPublisher
from app.window import WindowResult


class FakeProducerClient:
    """Mesmo papel de fakeProduceClient no Go: grava as chamadas a
    produce() para o teste inspecionar, sem precisar de um broker
    real."""

    def __init__(self) -> None:
        self.calls: list[dict] = []

    def produce(self, topic, key=None, value=None, **kwargs):
        self.calls.append({"topic": topic, "key": key, "value": value})

    def poll(self, timeout=0):
        return 0


def make_window_result() -> WindowResult:
    return WindowResult(
        zone_id="zone-042",
        window_start=datetime(2026, 6, 23, 14, 25, tzinfo=timezone.utc),
        window_end=datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc),
        average_soil_moisture_percent=31.2,
        rain_probability_percent=80,
    )


def test_publish_sends_to_irrigation_decision_topic():
    client = FakeProducerClient()
    publisher = DecisionPublisher(client=client)

    publisher.publish(make_window_result(), Decision(decision="START_IRRIGATION", confidence_score=0.74))

    assert len(client.calls) == 1
    assert client.calls[0]["topic"] == IRRIGATION_DECISION_TOPIC


def test_publish_uses_zone_id_as_partition_key():
    client = FakeProducerClient()
    publisher = DecisionPublisher(client=client)

    publisher.publish(make_window_result(), Decision(decision="START_IRRIGATION", confidence_score=0.74))

    assert client.calls[0]["key"] == b"zone-042"


def test_publish_wraps_payload_in_envelope_matching_readme_contract():
    client = FakeProducerClient()
    publisher = DecisionPublisher(client=client)

    publisher.publish(make_window_result(), Decision(decision="START_IRRIGATION", confidence_score=0.74))

    envelope = json.loads(client.calls[0]["value"])
    assert envelope["eventType"] == "IrrigationDecisionCalculated"
    assert envelope["producer"] == "irrigation-prediction"

    payload = envelope["payload"]
    assert payload["zoneId"] == "zone-042"
    assert payload["decision"] == "START_IRRIGATION"
    assert payload["windowStart"] == "2026-06-23T14:25:00Z"
    assert payload["windowEnd"] == "2026-06-23T14:30:00Z"
    assert payload["averageSoilMoisturePercent"] == 31.2
    assert payload["rainProbabilityPercent"] == 80
    assert payload["confidenceScore"] == 0.74
    assert payload["modelVersion"] == "v1"
