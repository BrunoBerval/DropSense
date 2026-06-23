package ingestion

import (
	"errors"
	"time"
)

// SoilReading is the structurally decoded representation of a single
// sensor payload — built by the HTTP layer after JSON decoding
// succeeds, but before any business validation runs.
type SoilReading struct {
	SensorID               string
	ZoneID                 string
	SoilMoisturePercent    float64
	SoilTemperatureCelsius float64
	MeasuredAt             time.Time
}

// Sentinel errors: callers (HTTP handler, tests) can branch on *which*
// rule failed with errors.Is, instead of parsing an error message.
var (
	ErrMissingSensorID        = errors.New("sensorId is required")
	ErrMissingZoneID          = errors.New("zoneId is required")
	ErrMoistureOutOfRange     = errors.New("soilMoisturePercent must be between 0 and 100")
	ErrTemperatureImplausible = errors.New("soilTemperatureCelsius is outside a plausible range")
)

const (
	// Faixa deliberadamente generosa: o objetivo aqui não é validar
	// agronomia (isso é problema do Core Domain, em C#) - é só
	// descartar leitura fisicamente impossível antes que ela entre
	// no pipeline e consuma rede/armazenamento.
	minPlausibleSoilTemperatureCelsius = -10.0
	maxPlausibleSoilTemperatureCelsius = 60.0
)

// Validate enforces the "fail fast at the boundary" rule defined for
// this service: anything that violates the laws of physics, or is
// missing the identifiers needed to route it, gets rejected here -
// before it ever reaches a channel, a worker, or Kafka.
func (r SoilReading) Validate() error {
	if r.SensorID == "" {
		return ErrMissingSensorID
	}
	if r.ZoneID == "" {
		return ErrMissingZoneID
	}
	if r.SoilMoisturePercent < 0 || r.SoilMoisturePercent > 100 {
		return ErrMoistureOutOfRange
	}
	if r.SoilTemperatureCelsius < minPlausibleSoilTemperatureCelsius ||
		r.SoilTemperatureCelsius > maxPlausibleSoilTemperatureCelsius {
		return ErrTemperatureImplausible
	}
	return nil
}
