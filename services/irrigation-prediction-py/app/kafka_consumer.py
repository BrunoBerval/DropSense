"""Consome um tópico em loop, numa thread própria, despachando cada
mensagem decodificada para on_message.

Mesma forma de Pipeline.worker no Go: roda em background, processa
até ctx ser cancelado - aqui, até stop_event ser sinalizado. O client
real (confluent_kafka.Consumer) é injetável via parâmetro `client`,
pelo mesmo motivo de produceClient no Go: permite testar o loop de
despacho sem broker nenhum.
"""

from __future__ import annotations

import logging
import threading
from collections.abc import Callable

logger = logging.getLogger(__name__)


class TopicConsumer:
    def __init__(
        self,
        topic: str,
        on_message: Callable[[bytes], None],
        brokers: str | None = None,
        group_id: str | None = None,
        client=None,
    ):
        if client is not None:
            self._consumer = client
        else:
            from confluent_kafka import Consumer

            self._consumer = Consumer(
                {
                    "bootstrap.servers": brokers,
                    "group.id": group_id,
                    "auto.offset.reset": "earliest",
                }
            )
        self._topic = topic
        self._on_message = on_message
        self._stop_event = threading.Event()
        self._thread: threading.Thread | None = None

    def start(self) -> None:
        self._consumer.subscribe([self._topic])
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()

    def _run(self) -> None:
        while not self._stop_event.is_set():
            msg = self._consumer.poll(timeout=1.0)
            if msg is None:
                continue
            if msg.error():
                logger.error("consumer error on topic %s: %s", self._topic, msg.error())
                continue
            try:
                self._on_message(msg.value())
            except Exception:
                logger.exception("failed to handle message from topic %s", self._topic)

    def stop(self) -> None:
        self._stop_event.set()
        if self._thread is not None:
            self._thread.join(timeout=5)
        self._consumer.close()
