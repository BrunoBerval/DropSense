package weather

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeFetcher e fakePublisher seguem o mesmo padrão de fake já usado
// no resto do projeto (fakeSubmitter em handler_test.go,
// recordingPublisher em pipeline_test.go): dublês simples, sem
// framework de mock, protegidos por mutex porque o Scheduler roda em
// goroutine própria.
type fakeFetcher struct {
	mu        sync.Mutex
	responses map[string]Forecast
	errs      map[string]error
}

func (f *fakeFetcher) FetchForecast(_ context.Context, zone Zone, _ int) (Forecast, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.errs[zone.ID]; ok {
		return Forecast{}, err
	}
	return f.responses[zone.ID], nil
}

type fakePublisher struct {
	mu        sync.Mutex
	published []string // zoneIDs publicados, na ordem em que chegaram
}

func (p *fakePublisher) PublishWeatherForecast(_ context.Context, zoneID string, _ Forecast) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.published = append(p.published, zoneID)
	return nil
}

func (p *fakePublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.published)
}

func (p *fakePublisher) contains(zoneID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, id := range p.published {
		if id == zoneID {
			return true
		}
	}
	return false
}

// waitUntil polla condition() até retornar true ou o timeout expirar -
// mesmo helper de pipeline_test.go, necessário porque o Scheduler
// publica de forma assíncrona.
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

func TestScheduler_OnStart_PublishesImmediately(t *testing.T) {
	fetcher := &fakeFetcher{responses: map[string]Forecast{"zone-042": {RainProbabilityPercent: 80}}}
	publisher := &fakePublisher{}
	// interval longo de propósito: se o primeiro publish só acontecer
	// no primeiro tick (1h), o teste estouraria o timeout - isso prova
	// que Start() não espera o Ticker para o primeiro ciclo.
	scheduler := NewScheduler(fetcher, publisher, []Zone{{ID: "zone-042"}}, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	waitUntil(t, time.Second, func() bool { return publisher.count() == 1 })

	if !publisher.contains("zone-042") {
		t.Error("expected zone-042 to have been published")
	}
}

func TestScheduler_PublishesForEveryZone(t *testing.T) {
	zones := []Zone{{ID: "zone-042"}, {ID: "zone-043"}}
	fetcher := &fakeFetcher{responses: map[string]Forecast{
		"zone-042": {RainProbabilityPercent: 10},
		"zone-043": {RainProbabilityPercent: 90},
	}}
	publisher := &fakePublisher{}
	scheduler := NewScheduler(fetcher, publisher, zones, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	waitUntil(t, time.Second, func() bool { return publisher.count() == 2 })

	if !publisher.contains("zone-042") || !publisher.contains("zone-043") {
		t.Errorf("expected both zones published, got %v", publisher.published)
	}
}

func TestScheduler_WhenFetchFailsForOneZone_StillPublishesOthers(t *testing.T) {
	zones := []Zone{{ID: "zone-broken"}, {ID: "zone-042"}}
	fetcher := &fakeFetcher{
		responses: map[string]Forecast{"zone-042": {RainProbabilityPercent: 50}},
		errs:      map[string]error{"zone-broken": errors.New("open-meteo unavailable")},
	}
	publisher := &fakePublisher{}
	scheduler := NewScheduler(fetcher, publisher, zones, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	waitUntil(t, time.Second, func() bool { return publisher.count() == 1 })

	if publisher.contains("zone-broken") {
		t.Error("expected zone-broken NOT to have been published")
	}
	if !publisher.contains("zone-042") {
		t.Error("expected zone-042 to have been published despite the other zone failing")
	}
}

func TestScheduler_RepeatsOnInterval(t *testing.T) {
	fetcher := &fakeFetcher{responses: map[string]Forecast{"zone-042": {}}}
	publisher := &fakePublisher{}
	scheduler := NewScheduler(fetcher, publisher, []Zone{{ID: "zone-042"}}, 20*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scheduler.Start(ctx)

	waitUntil(t, time.Second, func() bool { return publisher.count() >= 3 })
}

func TestScheduler_StopsAfterContextCancelled(t *testing.T) {
	fetcher := &fakeFetcher{responses: map[string]Forecast{"zone-042": {}}}
	publisher := &fakePublisher{}
	scheduler := NewScheduler(fetcher, publisher, []Zone{{ID: "zone-042"}}, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)

	waitUntil(t, time.Second, func() bool { return publisher.count() >= 1 })
	cancel() // simula shutdown gracioso (SIGTERM)

	countAtCancel := publisher.count()
	time.Sleep(100 * time.Millisecond)

	if publisher.count() != countAtCancel {
		t.Fatalf("expected no more publishes after cancel: had %d, now %d", countAtCancel, publisher.count())
	}
}
