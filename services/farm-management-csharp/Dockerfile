# syntax=docker/dockerfile:1
#
# Build multi-stage, mesmo espirito dos Dockerfiles do Go e do
# Python: o estagio builder restaura+publica com o SDK completo; o
# estagio final so leva o runtime ASP.NET Core (bem mais leve que o
# SDK) + o que foi publicado.

FROM mcr.microsoft.com/dotnet/sdk:10.0 AS builder
WORKDIR /src

# Copia so o .csproj primeiro - cache de layer do restore,
# independente do codigo mudar depois. Mesmo truque do
# "COPY go.mod go.sum* ./" no Dockerfile do Go.
COPY src/FarmManagement.Api/FarmManagement.Api.csproj src/FarmManagement.Api/
RUN dotnet restore src/FarmManagement.Api/FarmManagement.Api.csproj

COPY src/FarmManagement.Api/ src/FarmManagement.Api/
RUN dotnet publish src/FarmManagement.Api/FarmManagement.Api.csproj -c Release -o /app --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:10.0

# curl só para o HEALTHCHECK do docker-compose - a imagem base do
# ASP.NET Core não vem com nenhuma ferramenta HTTP de linha de
# comando por padrao.
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app .

EXPOSE 8080

ENTRYPOINT ["dotnet", "FarmManagement.Api.dll"]
