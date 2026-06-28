// Package csvsource lê o CSV gerado pelo script Python do Colab
// (gerar_sensores_dropsense.py) e agrupa as leituras por tick
// simulado.
package csvsource

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Reading é uma leitura de domínio, sem nada de fio - sem
// measuredAt, de propósito: carimbar o instante real do envio é
// responsabilidade de quem manda a requisição (internal/sender), não
// de quem lê o CSV. Mesma separação de ingestion.SoilReading no
// telemetry-ingestion.
type Reading struct {
	SensorID               string
	ZoneID                 string
	SoilMoisturePercent    float64
	SoilTemperatureCelsius float64
}

// expectedHeader trava o formato exato gerado pelo script Python -
// se o CSV não bater (ex.: colunas em outra ordem, arquivo errado),
// falha alto e claro aqui, em vez de mandar dado errado pro
// telemetry-ingestion silenciosamente.
var expectedHeader = []string{
	"sensorId", "zoneId", "soilMoisturePercent", "soilTemperatureCelsius", "_passoSimulado",
}

// LoadTicks lê o CSV inteiro e agrupa as leituras pela 5ª coluna
// (_passoSimulado). O gerador já escreve o arquivo nessa ordem (todo
// mundo do mesmo tick em linhas contíguas), então basta detectar
// quando esse valor muda. Cada elemento do retorno é "todos os
// sensores que mediram nesse instante simulado".
func LoadTicks(path string) ([][]Reading, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("csvsource: failed to open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("csvsource: failed to read header: %w", err)
	}
	if err := validateHeader(header); err != nil {
		return nil, err
	}

	var ticks [][]Reading
	var currentTick []Reading
	currentStep := ""
	first := true

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csvsource: failed to read row: %w", err)
		}

		step := record[4]
		if first || step != currentStep {
			if !first {
				ticks = append(ticks, currentTick)
			}
			currentTick = nil
			currentStep = step
			first = false
		}

		moisture, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return nil, fmt.Errorf("csvsource: invalid soilMoisturePercent %q: %w", record[2], err)
		}
		temperature, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			return nil, fmt.Errorf("csvsource: invalid soilTemperatureCelsius %q: %w", record[3], err)
		}

		currentTick = append(currentTick, Reading{
			SensorID:               record[0],
			ZoneID:                 record[1],
			SoilMoisturePercent:    moisture,
			SoilTemperatureCelsius: temperature,
		})
	}

	if len(currentTick) > 0 {
		ticks = append(ticks, currentTick)
	}

	return ticks, nil
}

func validateHeader(header []string) error {
	if len(header) != len(expectedHeader) {
		return fmt.Errorf("csvsource: expected %d columns, got %d (header: %v)", len(expectedHeader), len(header), header)
	}
	for i, want := range expectedHeader {
		if header[i] != want {
			return fmt.Errorf("csvsource: expected column %d to be %q, got %q", i, want, header[i])
		}
	}
	return nil
}
