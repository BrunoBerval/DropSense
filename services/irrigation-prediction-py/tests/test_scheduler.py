import threading
import time
from datetime import datetime, timedelta, timezone

import pytest

from app.decision import Decision
from app.scheduler import DecisionScheduler
from app.window import WindowResult


class FakeAggregator:
    """Mesmo papel de fakeFetcher no Go (weather/scheduler_test.go):
    devolve resultados pré-canned por zoneId, sem nenhum dado real."""

    def __init__(self, windows: dict[str, WindowResult]):
        self._windows = windows
        self.lock = threading.Lock()

    def known_zones(self) -> list[str]:
        with self.lock:
            return list(self._windows.keys())

    def compute_window(self, zone_id: str, now) -> WindowResult | None:
        with self.lock:
            return self._windows.get(zone_id)


class FakePublisher:
    """Mesmo papel de fakePublisher no Go: grava zoneIds publicados."""

    def __init__(self) -> None:
        self.lock = threading.Lock()
        self.published: list[str] = []

    def publish(self, window: WindowResult, decision: Decision) -> None:
        with self.lock:
            self.published.append(window.zone_id)

    def count(self) -> int:
        with self.lock:
            return len(self.published)

    def contains(self, zone_id: str) -> bool:
        with self.lock:
            return zone_id in self.published


def make_window(zone_id: str, moisture: float = 20.0, rain: int = 0) -> WindowResult:
    now = datetime(2026, 6, 23, 14, 30, tzinfo=timezone.utc)
    return WindowResult(
        zone_id=zone_id,
        window_start=now - timedelta(minutes=5),
        window_end=now,
        average_soil_moisture_percent=moisture,
        rain_probability_percent=rain,
    )


def wait_until(condition, timeout_seconds: float = 1.0) -> None:
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        if condition():
            return
        time.sleep(0.005)
    pytest.fail("condition not met within timeout")


def test_scheduler_publishes_immediately_on_start():
    aggregator = FakeAggregator({"zone-042": make_window("zone-042")})
    publisher = FakePublisher()
    # interval longo de propósito: se o primeiro publish só
    # acontecesse no primeiro tick, o teste estouraria o timeout -
    # prova que start() não espera o intervalo para o primeiro ciclo.
    scheduler = DecisionScheduler(aggregator, publisher, interval=timedelta(hours=1))

    scheduler.start()
    try:
        wait_until(lambda: publisher.count() == 1)
        assert publisher.contains("zone-042")
    finally:
        scheduler.stop()


def test_scheduler_publishes_for_every_known_zone():
    aggregator = FakeAggregator(
        {
            "zone-042": make_window("zone-042"),
            "zone-099": make_window("zone-099"),
        }
    )
    publisher = FakePublisher()
    scheduler = DecisionScheduler(aggregator, publisher, interval=timedelta(hours=1))

    scheduler.start()
    try:
        wait_until(lambda: publisher.count() == 2)
        assert publisher.contains("zone-042")
        assert publisher.contains("zone-099")
    finally:
        scheduler.stop()


def test_scheduler_skips_zones_with_no_window_yet():
    aggregator = FakeAggregator({"zone-sem-dados": None})
    publisher = FakePublisher()
    scheduler = DecisionScheduler(aggregator, publisher, interval=timedelta(hours=1))

    scheduler.start()
    try:
        time.sleep(0.1)
        assert publisher.count() == 0
    finally:
        scheduler.stop()


def test_scheduler_repeats_on_interval():
    aggregator = FakeAggregator({"zone-042": make_window("zone-042")})
    publisher = FakePublisher()
    scheduler = DecisionScheduler(aggregator, publisher, interval=timedelta(milliseconds=20))

    scheduler.start()
    try:
        wait_until(lambda: publisher.count() >= 3)
    finally:
        scheduler.stop()


def test_scheduler_stops_after_stop_called():
    aggregator = FakeAggregator({"zone-042": make_window("zone-042")})
    publisher = FakePublisher()
    scheduler = DecisionScheduler(aggregator, publisher, interval=timedelta(milliseconds=10))

    scheduler.start()
    wait_until(lambda: publisher.count() >= 1)
    scheduler.stop()

    count_at_stop = publisher.count()
    time.sleep(0.1)

    assert publisher.count() == count_at_stop
