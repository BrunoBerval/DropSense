import threading
import time

import pytest

from app.kafka_consumer import TopicConsumer


class FakeMessage:
    def __init__(self, value: bytes, error=None):
        self._value = value
        self._error = error

    def value(self) -> bytes:
        return self._value

    def error(self):
        return self._error


class FakeConsumerClient:
    """Mesmo papel de fakeProduceClient: simula o que o
    confluent_kafka.Consumer real faria, sem broker nenhum. poll()
    devolve mensagens de uma fila pré-armada, e None quando ela
    esvazia (igual o client real faz quando não há nada de novo)."""

    def __init__(self, messages: list) -> None:
        self.lock = threading.Lock()
        self._messages = list(messages)
        self.subscribed_topics: list[str] | None = None
        self.closed = False

    def subscribe(self, topics: list[str]) -> None:
        self.subscribed_topics = topics

    def poll(self, timeout: float = 1.0):
        with self.lock:
            if self._messages:
                return self._messages.pop(0)
        return None

    def close(self) -> None:
        self.closed = True


def wait_until(condition, timeout_seconds: float = 1.0) -> None:
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        if condition():
            return
        time.sleep(0.005)
    pytest.fail("condition not met within timeout")


def test_consumer_subscribes_to_the_configured_topic():
    client = FakeConsumerClient(messages=[])
    consumer = TopicConsumer(topic="telemetry.readings.v1", on_message=lambda raw: None, client=client)

    consumer.start()
    try:
        wait_until(lambda: client.subscribed_topics is not None)
        assert client.subscribed_topics == ["telemetry.readings.v1"]
    finally:
        consumer.stop()


def test_consumer_dispatches_message_value_to_on_message():
    received: list[bytes] = []
    client = FakeConsumerClient(messages=[FakeMessage(value=b'{"hello":"world"}')])
    consumer = TopicConsumer(topic="telemetry.readings.v1", on_message=received.append, client=client)

    consumer.start()
    try:
        wait_until(lambda: len(received) == 1)
        assert received[0] == b'{"hello":"world"}'
    finally:
        consumer.stop()


def test_consumer_skips_messages_with_error_without_calling_on_message():
    received: list[bytes] = []
    client = FakeConsumerClient(
        messages=[
            FakeMessage(value=b"", error="boom"),
            FakeMessage(value=b'{"ok":true}'),
        ]
    )
    consumer = TopicConsumer(topic="telemetry.readings.v1", on_message=received.append, client=client)

    consumer.start()
    try:
        wait_until(lambda: len(received) == 1)
        assert received == [b'{"ok":true}']
    finally:
        consumer.stop()


def test_consumer_does_not_crash_when_on_message_raises():
    def exploding_handler(raw: bytes) -> None:
        raise RuntimeError("boom")

    client = FakeConsumerClient(messages=[FakeMessage(value=b"x"), FakeMessage(value=b"y")])
    consumer = TopicConsumer(topic="telemetry.readings.v1", on_message=exploding_handler, client=client)

    consumer.start()
    try:
        # se a thread tivesse morrido na primeira exceção, a fila
        # nunca seria esvaziada.
        wait_until(lambda: len(client._messages) == 0)
    finally:
        consumer.stop()


def test_consumer_closes_client_on_stop():
    client = FakeConsumerClient(messages=[])
    consumer = TopicConsumer(topic="telemetry.readings.v1", on_message=lambda raw: None, client=client)

    consumer.start()
    consumer.stop()

    assert client.closed is True
