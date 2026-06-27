from app.decision import MODEL_VERSION, SKIP_IRRIGATION, START_IRRIGATION, decide


def test_decide_skips_when_moisture_already_comfortable():
    decision = decide(average_soil_moisture_percent=40.0, rain_probability_percent=5)

    assert decision.decision == SKIP_IRRIGATION


def test_decide_skips_when_rain_probability_is_high_even_with_low_moisture():
    decision = decide(average_soil_moisture_percent=20.0, rain_probability_percent=85)

    assert decision.decision == SKIP_IRRIGATION


def test_decide_starts_irrigation_when_dry_and_unlikely_to_rain():
    decision = decide(average_soil_moisture_percent=20.0, rain_probability_percent=10)

    assert decision.decision == START_IRRIGATION


def test_decide_confidence_score_is_between_zero_and_one():
    decision = decide(average_soil_moisture_percent=20.0, rain_probability_percent=10)

    assert 0.0 <= decision.confidence_score <= 1.0


def test_model_version_is_v1():
    # v1 == regra simples, de propósito - o README já avisa que o
    # modelo pode ser "simplificado para fins do projeto". modelVersion
    # existe no contrato exatamente para permitir trocar isso por algo
    # com scikit-learn depois, sem quebrar quem consome o evento.
    assert MODEL_VERSION == "v1"
