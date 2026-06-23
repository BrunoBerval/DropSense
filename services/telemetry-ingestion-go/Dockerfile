# --- Stage 1: build ---
# Imagem com o toolchain completo do Go (~800MB), usada SO pra compilar.
# Ela nao vai pra imagem final - e descartada depois do build.
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copia so os arquivos de dependencia primeiro: o Docker cacheia essa
# camada, entao "go mod download" so roda de novo se go.mod/go.sum
# mudarem - nao a cada alteracao de codigo.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: binario estatico, sem depender de libc do sistema.
RUN CGO_ENABLED=0 GOOS=linux go build -o /telemetry-server ./cmd/server

# --- Stage 2: runtime ---
# alpine (nao "scratch"): este servico vai fazer chamada HTTPS pra API
# de clima no proximo passo, e precisa dos certificados CA pra validar
# TLS. "scratch" nao tem isso. alpine e minima (~5-8MB) e ja resolve.
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /telemetry-server /telemetry-server

EXPOSE 8080

ENTRYPOINT ["/telemetry-server"]
