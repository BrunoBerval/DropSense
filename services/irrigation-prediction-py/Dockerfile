# syntax=docker/dockerfile:1
#
# Build multi-stage: o estagio "builder" instala as dependencias num
# prefixo isolado (/install), e o estagio final copia só esse
# resultado - sem cache do pip, sem ferramenta de build sobrando na
# imagem final. Mesmo espirito do Dockerfile do telemetry-ingestion-go
# (builder separado do runtime), só que aqui não há binário compilado
# para copiar - o "artefato" é o conjunto de pacotes instalados.
#
# IMPORTANTE: usamos "slim" (Debian/glibc), não "alpine" (musl).
# O pacote confluent-kafka é distribuído como wheel manylinux
# (glibc) - não existe wheel para musl, então em Alpine o pip cairia
# para compilar a partir do source, exigindo libradkafka-dev e um
# compilador C instalados manualmente. Em slim, o wheel pré-compilado
# funciona direto, sem nenhuma dependência de sistema extra.

FROM python:3.12-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

FROM python:3.12-slim
WORKDIR /app
COPY --from=builder /install /usr/local
COPY app ./app

EXPOSE 8000

# Sem --workers: por padrão o Uvicorn roda um processo só. Múltiplos
# workers duplicariam os consumidores Kafka (cada processo abriria
# sua própria thread de consumo) - se algum dia for preciso escalar,
# o caminho correto é múltiplas réplicas do container compartilhando
# o mesmo consumer group (Kafka distribui as partições entre elas),
# não múltiplos workers Uvicorn dentro do mesmo container.
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
