"""Testa só o que é seguro testar sem broker: a rota /healthz em si.
De propósito, NÃO instancia TestClient com o lifespan ativo aqui -
isso conectaria de verdade no Kafka (via DecisionPublisher/
TopicConsumer reais), o que pertence à validação manual com
docker-compose, não à suíte unitária - mesma fronteira que já
seguimos no Go (testes unitários usam fakes; a validação real é
feita rodando a stack)."""

from app.main import app, healthz


def test_healthz_function_returns_ok_status():
    assert healthz() == {"status": "ok"}


def test_healthz_route_is_registered():
    paths = [route.path for route in app.routes]
    assert "/healthz" in paths
