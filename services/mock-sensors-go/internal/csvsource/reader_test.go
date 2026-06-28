package csvsource

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "readings.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}
	return path
}

func TestLoadTicks_GroupsConsecutiveRowsBySimulatedStep(t *testing.T) {
	content := "sensorId,zoneId,soilMoisturePercent,soilTemperatureCelsius,_passoSimulado\n" +
		"sensor-001-001,zone-001,40.0,20.0,0\n" +
		"sensor-002-001,zone-002,30.0,21.0,0\n" +
		"sensor-001-001,zone-001,39.5,20.1,1\n"

	path := writeTempCSV(t, content)

	ticks, err := LoadTicks(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(ticks) != 2 {
		t.Fatalf("expected 2 ticks, got %d", len(ticks))
	}
	if len(ticks[0]) != 2 {
		t.Fatalf("expected 2 readings in first tick, got %d", len(ticks[0]))
	}
	if len(ticks[1]) != 1 {
		t.Fatalf("expected 1 reading in second tick, got %d", len(ticks[1]))
	}
}

func TestLoadTicks_ParsesAllFieldsCorrectly(t *testing.T) {
	content := "sensorId,zoneId,soilMoisturePercent,soilTemperatureCelsius,_passoSimulado\n" +
		"sensor-001-001,zone-001,40.5,20.3,0\n"

	path := writeTempCSV(t, content)

	ticks, err := LoadTicks(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	reading := ticks[0][0]
	if reading.SensorID != "sensor-001-001" {
		t.Errorf("SensorID = %q, want %q", reading.SensorID, "sensor-001-001")
	}
	if reading.ZoneID != "zone-001" {
		t.Errorf("ZoneID = %q, want %q", reading.ZoneID, "zone-001")
	}
	if reading.SoilMoisturePercent != 40.5 {
		t.Errorf("SoilMoisturePercent = %v, want 40.5", reading.SoilMoisturePercent)
	}
	if reading.SoilTemperatureCelsius != 20.3 {
		t.Errorf("SoilTemperatureCelsius = %v, want 20.3", reading.SoilTemperatureCelsius)
	}
}

func TestLoadTicks_RejectsWrongHeader(t *testing.T) {
	content := "foo,bar\n1,2\n"
	path := writeTempCSV(t, content)

	if _, err := LoadTicks(path); err == nil {
		t.Fatal("expected an error for a CSV with the wrong header, got nil")
	}
}

func TestLoadTicks_RejectsInvalidMoisture(t *testing.T) {
	content := "sensorId,zoneId,soilMoisturePercent,soilTemperatureCelsius,_passoSimulado\n" +
		"sensor-001-001,zone-001,not-a-number,20.0,0\n"
	path := writeTempCSV(t, content)

	if _, err := LoadTicks(path); err == nil {
		t.Fatal("expected an error for an invalid soilMoisturePercent, got nil")
	}
}

func TestLoadTicks_RejectsInvalidTemperature(t *testing.T) {
	content := "sensorId,zoneId,soilMoisturePercent,soilTemperatureCelsius,_passoSimulado\n" +
		"sensor-001-001,zone-001,40.0,not-a-number,0\n"
	path := writeTempCSV(t, content)

	if _, err := LoadTicks(path); err == nil {
		t.Fatal("expected an error for an invalid soilTemperatureCelsius, got nil")
	}
}

func TestLoadTicks_FileNotFound_ReturnsError(t *testing.T) {
	if _, err := LoadTicks("/nonexistent/path.csv"); err == nil {
		t.Fatal("expected an error for a missing file, got nil")
	}
}

func TestLoadTicks_OnlyHeader_ReturnsNoTicks(t *testing.T) {
	content := "sensorId,zoneId,soilMoisturePercent,soilTemperatureCelsius,_passoSimulado\n"
	path := writeTempCSV(t, content)

	ticks, err := LoadTicks(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ticks) != 0 {
		t.Fatalf("expected 0 ticks, got %d", len(ticks))
	}
}
