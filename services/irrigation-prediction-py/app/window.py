"""Agregação de leituras de solo por zona, numa janela de tempo
deslizante (tumbling, recalculada a cada compute_window) - "agrupa as
leituras por Zona em janelas de tempo (por exemplo, a média de
umidade da Zona A nos últimos 5 minutos)", como o README descreve
esse serviço fazendo, na Parte 2.

Mantido inteiramente em memória nesta fase, de propósito - mesmo
princípio já seguido no Go (StdoutPublisher, depois Kafka real, sem
Postgres até haver necessidade real): persistência entra quando
houver motivo concreto, não antecipada.
"""

from __future__ import annotations

import threading
from collections import defaultdict
from dataclasses import dataclass
from datetime import datetime, timedelta

from app.events import SoilReadingPayload, WeatherForecastPayload


@dataclass(frozen=True)
class WindowResult:
    """Resultado agregado de uma zona para uma janela - a forma
    "pronta para decidir", depois de processar as leituras brutas."""

    zone_id: str
    window_start: datetime
    window_end: datetime
    average_soil_moisture_percent: float
    rain_probability_percent: int


class WindowAggregator:
    """Guarda, por zona: as leituras de solo recentes (descartando o
    que sai da janela a cada consulta) e a previsão do tempo mais
    recente conhecida. Thread-safe porque é escrito pela thread do
    consumidor Kafka e lido pela thread do scheduler de decisão, ao
    mesmo tempo."""

    def __init__(self, window_size: timedelta):
        self._window_size = window_size
        self._readings: dict[str, list[tuple[datetime, float]]] = defaultdict(list)
        self._latest_forecast: dict[str, WeatherForecastPayload] = {}
        self._lock = threading.Lock()

    def add_soil_reading(self, payload: SoilReadingPayload, received_at: datetime) -> None:
        with self._lock:
            self._readings[payload.zone_id].append((received_at, payload.soil_moisture_percent))
            self._evict_old(payload.zone_id, received_at)

    def set_weather_forecast(self, payload: WeatherForecastPayload) -> None:
        with self._lock:
            self._latest_forecast[payload.zone_id] = payload

    def compute_window(self, zone_id: str, now: datetime) -> WindowResult | None:
        """Calcula o resultado agregado da zona até `now`. Retorna
        None se não há nenhuma leitura recente para essa zona - nesse
        caso, não há nada útil a publicar (ver DecisionScheduler)."""
        with self._lock:
            self._evict_old(zone_id, now)
            readings = self._readings.get(zone_id, [])
            if not readings:
                return None

            average_moisture = sum(value for _, value in readings) / len(readings)
            forecast = self._latest_forecast.get(zone_id)
            rain_probability = forecast.rain_probability_percent if forecast else 0

            return WindowResult(
                zone_id=zone_id,
                window_start=now - self._window_size,
                window_end=now,
                average_soil_moisture_percent=average_moisture,
                rain_probability_percent=rain_probability,
            )

    def known_zones(self) -> list[str]:
        """Zonas com pelo menos uma leitura de solo já registrada -
        usado pelo DecisionScheduler para saber quais zonas processar
        a cada ciclo."""
        with self._lock:
            return [zone_id for zone_id, readings in self._readings.items() if readings]

    def _evict_old(self, zone_id: str, now: datetime) -> None:
        cutoff = now - self._window_size
        self._readings[zone_id] = [
            (timestamp, value) for timestamp, value in self._readings[zone_id] if timestamp >= cutoff
        ]
