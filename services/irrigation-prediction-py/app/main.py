"""Ponto de entrada do serviço - FastAPI só para /healthz (e, no
futuro, métricas Prometheus); o trabalho real é o consumidor Kafka +
scheduler de decisão, rodando em threads de fundo a partir do
lifespan.

Mesma forma do main.go: lá, pipeline.Start(ctx) e
weatherScheduler.Start(ctx) rodam em goroutines enquanto
server.ListenAndServe() ocupa a thread principal; aqui, os
consumidores e o scheduler rodam em threads enquanto o Uvicorn ocupa
o processo principal servindo HTTP.
"""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.config import get_settings
from app.events import SOIL_READING_TOPIC, WEATHER_FORECAST_TOPIC
from app.kafka_consumer import TopicConsumer
from app.kafka_producer import DecisionPublisher
from app.router import handle_message
from app.scheduler import DecisionScheduler
from app.window import WindowAggregator

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    logger.info("starting irrigation-prediction (window=%s, decision_interval=%s)", settings.window_size, settings.decision_interval)

    aggregator = WindowAggregator(window_size=settings.window_size)
    publisher = DecisionPublisher(brokers=settings.kafka_brokers)

    soil_consumer = TopicConsumer(
        topic=SOIL_READING_TOPIC,
        brokers=settings.kafka_brokers,
        group_id=settings.consumer_group,
        on_message=lambda raw: handle_message(raw, aggregator),
    )
    weather_consumer = TopicConsumer(
        topic=WEATHER_FORECAST_TOPIC,
        brokers=settings.kafka_brokers,
        group_id=settings.consumer_group,
        on_message=lambda raw: handle_message(raw, aggregator),
    )
    scheduler = DecisionScheduler(aggregator, publisher, interval=settings.decision_interval)

    soil_consumer.start()
    weather_consumer.start()
    scheduler.start()

    app.state.aggregator = aggregator

    yield

    logger.info("shutdown signal received, stopping background threads...")
    scheduler.stop()
    soil_consumer.stop()
    weather_consumer.stop()
    publisher.close()


app = FastAPI(lifespan=lifespan)


@app.get("/healthz")
def healthz():
    return {"status": "ok"}
