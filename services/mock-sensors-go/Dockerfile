# syntax=docker/dockerfile:1
#
# Build multi-stage, mesmo formato do Dockerfile do telemetry-ingestion-go.
# Sem ca-certificates aqui, diferente do irmao: este servico so faz
# HTTP simples (sem TLS) contra outro container na mesma rede do
# compose - nao tem nenhum certificado pra validar.

FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /mock-sensors ./cmd/mock-sensors

FROM alpine:3.19
COPY --from=builder /mock-sensors /mock-sensors

ENTRYPOINT ["/mock-sensors"]
