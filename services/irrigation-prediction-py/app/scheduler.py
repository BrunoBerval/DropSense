"""Dispara o cálculo e a publicação de decisões de irrigação, em
intervalos regulares, para cada zona conhecida.

Mesma forma do weather.Scheduler no Go: roda em thread própria
(lá, goroutine), dispara um ciclo imediatamente ao iniciar (não
espera o primeiro tick), repete por intervalo, e para de forma
limpa quando stop() é chamado - threading.Event.wait(timeout) faz
aqui o mesmo papel de "select { case <-ctx.Done(): ...; case
<-ticker.C: ... }" no Go.
"""

from __future__ import annotations

import logging
import threading
from datetime import datetime, timedelta, timezone
from typing import Protocol

from app.decision import decide
from app.window import WindowResult

logger = logging.getLogger(__name__)


class _Aggregator(Protocol):
    def known_zones(self) -> list[str]: ...
    def compute_window(self, zone_id: str, now: datetime) -> WindowResult | None: ...


class _Publisher(Protocol):
    def publish(self, window: WindowResult, decision) -> None: ...


class DecisionScheduler:
    def __init__(self, aggregator: _Aggregator, publisher: _Publisher, interval: timedelta):
        self._aggregator = aggregator
        self._publisher = publisher
        self._interval = interval
        self._stop_event = threading.Event()
        self._thread: threading.Thread | None = None

    def start(self) -> None:
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()

    def _run(self) -> None:
        self._tick()
        while not self._stop_event.wait(timeout=self._interval.total_seconds()):
            self._tick()

    def _tick(self) -> None:
        now = datetime.now(timezone.utc)
        for zone_id in self._aggregator.known_zones():
            window = self._aggregator.compute_window(zone_id, now)
            if window is None:
                continue
            decision = decide(window.average_soil_moisture_percent, window.rain_probability_percent)
            try:
                self._publisher.publish(window, decision)
            except Exception:
                logger.exception("failed to publish decision for zone %s", zone_id)

    def stop(self) -> None:
        self._stop_event.set()
        if self._thread is not None:
            self._thread.join(timeout=5)
