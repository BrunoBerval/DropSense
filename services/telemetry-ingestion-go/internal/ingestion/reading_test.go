package ingestion

import (
	"errors"
	"testing"
	"time"
)

func validReading() SoilReading {
	return SoilReading{
		SensorID:               "sensor-04812",
		ZoneID:                 "zone-042",
		SoilMoisturePercent:    38.5,
		SoilTemperatureCelsius: 24.1,
		MeasuredAt:             time.Now(),
	}
}

func TestValidate_WithValidReading_ReturnsNoError(t *testing.T) {
	reading := validReading()

	if err := reading.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_WithMissingSensorID_ReturnsError(t *testing.T) {
	reading := validReading()
	reading.SensorID = ""

	if err := reading.Validate(); !errors.Is(err, ErrMissingSensorID) {
		t.Fatalf("expected ErrMissingSensorID, got %v", err)
	}
}

func TestValidate_WithMissingZoneID_ReturnsError(t *testing.T) {
	reading := validReading()
	reading.ZoneID = ""

	if err := reading.Validate(); !errors.Is(err, ErrMissingZoneID) {
		t.Fatalf("expected ErrMissingZoneID, got %v", err)
	}
}

func TestValidate_WithMoistureOutOfPhysicalRange_ReturnsError(t *testing.T) {
	invalidValues := []float64{-0.1, 100.1, -50, 250}

	for _, moisture := range invalidValues {
		reading := validReading()
		reading.SoilMoisturePercent = moisture

		if err := reading.Validate(); !errors.Is(err, ErrMoistureOutOfRange) {
			t.Errorf("moisture=%v: expected ErrMoistureOutOfRange, got %v", moisture, err)
		}
	}
}

func TestValidate_WithImplausibleTemperature_ReturnsError(t *testing.T) {
	invalidValues := []float64{-15, 70}

	for _, temp := range invalidValues {
		reading := validReading()
		reading.SoilTemperatureCelsius = temp

		if err := reading.Validate(); !errors.Is(err, ErrTemperatureImplausible) {
			t.Errorf("temp=%v: expected ErrTemperatureImplausible, got %v", temp, err)
		}
	}
}
