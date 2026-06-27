"""Modelo de decisão de irrigação - v1.

O README já avisa, na Parte 2: "aplica um modelo preditivo - mesmo que
simplificado para fins do projeto". Esta é essa versão simplificada:
duas regras de limiar, sem aprendizado de máquina. confidence_score é
uma heurística fixa por ramo, não uma probabilidade calculada - é
"o quão confiante esse modelo simples está na própria regra que
acabou de aplicar", não uma métrica estatística.

modelVersion="v1" no contrato publicado existe exatamente para
permitir trocar isto por um modelo de verdade (scikit-learn, como o
README cita no stack) depois, sem quebrar quem consome
IrrigationDecisionCalculated - o contrato (zoneId, decision,
confidenceScore, modelVersion, ...) não muda, só a lógica por trás dele.
"""

from __future__ import annotations

from dataclasses import dataclass

MODEL_VERSION = "v1"

START_IRRIGATION = "START_IRRIGATION"
SKIP_IRRIGATION = "SKIP_IRRIGATION"

# Limiares deliberadamente simples e nomeados, para serem fáceis de
# ajustar/discutir - não há agronomia real por trás destes números
# (mesma postura do reading.go no Go: "a faixa é deliberadamente
# generosa, o objetivo aqui não é validar agronomia").
MOISTURE_COMFORTABLE_THRESHOLD_PERCENT = 35.0
RAIN_PROBABILITY_SKIP_THRESHOLD_PERCENT = 60


@dataclass(frozen=True)
class Decision:
    decision: str
    confidence_score: float


def decide(average_soil_moisture_percent: float, rain_probability_percent: int) -> Decision:
    """"A umidade atual está em 30%, mas há 80% de chance de chuva nas
    próximas 2 horas. Vale a pena irrigar agora?" - a pergunta que o
    README usa como exemplo, na Parte 2, é literalmente o que esta
    função responde.
    """
    if average_soil_moisture_percent >= MOISTURE_COMFORTABLE_THRESHOLD_PERCENT:
        return Decision(decision=SKIP_IRRIGATION, confidence_score=0.9)
    if rain_probability_percent >= RAIN_PROBABILITY_SKIP_THRESHOLD_PERCENT:
        return Decision(decision=SKIP_IRRIGATION, confidence_score=0.7)
    return Decision(decision=START_IRRIGATION, confidence_score=0.8)
