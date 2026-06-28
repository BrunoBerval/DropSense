package sender

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"dropsense/mock-sensors/internal/csvsource"
)

// fakeSender grava, em ordem, todo sensorId que recebeu - mesmo
// papel de fakePublisher/recordingPublisher já usados no resto do
// projeto. Protegido por mutex porque o Runner dispara uma goroutine
// por leitura dentro de cada tick.
type fakeSender struct {
	mu  sync.Mutex
	got []string
	err map[string]error
}

func (f *fakeSender) Send(_ context.Context, reading csvsource.Reading) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, reading.SensorID)
	if f.err != nil {
		if err, ok := f.err[reading.SensorID]; ok {
			return err
		}
	}
	return nil
}

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.got)
}

func (f *fakeSender) countOf(sensorID string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, id := range f.got {
		if id == sensorID {
			n++
		}
	}
	return n
}

func (f *fakeSender) contains(sensorID string) bool {
	return f.countOf(sensorID) > 0
}

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func tick(sensorIDs ...string) []csvsource.Reading {
	readings := make([]csvsource.Reading, len(sensorIDs))
	for i, id := range sensorIDs {
		readings[i] = csvsource.Reading{SensorID: id, ZoneID: "zone-001", SoilMoisturePercent: 40, SoilTemperatureCelsius: 20}
	}
	return readings
}

func TestRunner_SendsFirstTickImmediately(t *testing.T) {
	sender := &fakeSender{}
	// interval longo de propósito: se o primeiro envio só
	// acontecesse no primeiro tick do ticker, o teste estouraria o
	// timeout - prova que Run() não espera o intervalo para o
	// primeiro tick.
	runner := NewRunner(sender, [][]csvsource.Reading{tick("sensor-1", "sensor-2")}, time.Hour, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)

	waitUntil(t, time.Second, func() bool { return sender.count() == 2 })
	if !sender.contains("sensor-1") || !sender.contains("sensor-2") {
		t.Errorf("expected both sensors sent, got %v", sender.got)
	}
}

func TestRunner_AdvancesToNextTickOnInterval(t *testing.T) {
	sender := &fakeSender{}
	ticks := [][]csvsource.Reading{tick("sensor-1"), tick("sensor-2")}
	runner := NewRunner(sender, ticks, 20*time.Millisecond, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)

	waitUntil(t, time.Second, func() bool { return sender.contains("sensor-2") })
}

func TestRunner_WithoutLoop_StopsAfterLastTick(t *testing.T) {
	sender := &fakeSender{}
	runner := NewRunner(sender, [][]csvsource.Reading{tick("sensor-1")}, 10*time.Millisecond, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)

	waitUntil(t, time.Second, func() bool { return sender.count() == 1 })
	time.Sleep(100 * time.Millisecond) // tempo de sobra para um ciclo extra, se houvesse

	if sender.count() != 1 {
		t.Fatalf("expected exactly 1 send (sem loop, 1 tick só), got %d", sender.count())
	}
}

func TestRunner_WithLoop_RestartsFromFirstTick(t *testing.T) {
	sender := &fakeSender{}
	ticks := [][]csvsource.Reading{tick("sensor-1"), tick("sensor-2")}
	runner := NewRunner(sender, ticks, 10*time.Millisecond, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)

	waitUntil(t, time.Second, func() bool { return sender.countOf("sensor-1") >= 2 })
}

func TestRunner_OneSensorFailing_DoesNotStopOthersInTheSameTick(t *testing.T) {
	sender := &fakeSender{err: map[string]error{"sensor-broken": errors.New("boom")}}
	runner := NewRunner(sender, [][]csvsource.Reading{tick("sensor-broken", "sensor-ok")}, time.Hour, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)

	waitUntil(t, time.Second, func() bool { return sender.contains("sensor-ok") })
}

func TestRunner_StopsWhenContextCancelled(t *testing.T) {
	sender := &fakeSender{}
	// interval de 200ms (nao 10ms) de proposito: precisamos de uma
	// janela confortavel entre o cancel() e o proximo tick do ticker,
	// senao o select{ctx.Done(); ticker.C} poderia escolher
	// qualquer um dos dois quando os dois estao prontos quase ao
	// mesmo tempo - flakiness, nao bug de logica.
	runner := NewRunner(sender, [][]csvsource.Reading{tick("sensor-1")}, 200*time.Millisecond, true)

	ctx, cancel := context.WithCancel(context.Background())
	go runner.Run(ctx)

	waitUntil(t, time.Second, func() bool { return sender.count() >= 1 })
	cancel()

	countAtCancel := sender.count()
	time.Sleep(300 * time.Millisecond)

	if sender.count() != countAtCancel {
		t.Fatalf("expected no more sends after cancel: had %d, now %d", countAtCancel, sender.count())
	}
}

func TestRunner_EmptyTicks_ReturnsWithoutBlocking(t *testing.T) {
	sender := &fakeSender{}
	runner := NewRunner(sender, nil, time.Hour, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runner.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return for empty ticks")
	}
}
