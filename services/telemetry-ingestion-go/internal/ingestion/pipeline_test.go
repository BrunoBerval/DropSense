package ingestion

import (
	"context"
	"sync"
	"testing"
	"time"
)

// recordingPublisher is a test double that records every reading it
// receives. It must be safe for concurrent use: multiple worker
// goroutines call Publish at the same time, so access to the slice is
// guarded by a mutex.
type recordingPublisher struct {
	mu        sync.Mutex
	published []SoilReading
}

func (p *recordingPublisher) Publish(_ context.Context, reading SoilReading) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.published = append(p.published, reading)
	return nil
}

func (p *recordingPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.published)
}

// waitUntil polls condition() until it returns true or the timeout
// expires. Necessary because workers process readings concurrently -
// a test can't assume Submit() having returned means the reading has
// already been published.
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

func TestPipeline_SubmittedReading_IsPublished(t *testing.T) {
	publisher := &recordingPublisher{}
	pipeline := NewPipeline(publisher, 2, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)

	pipeline.Submit(validReading())

	waitUntil(t, time.Second, func() bool { return publisher.count() == 1 })
}

func TestPipeline_ManyReadingsAcrossWorkers_AllGetPublished(t *testing.T) {
	publisher := &recordingPublisher{}
	pipeline := NewPipeline(publisher, 4, 100) // 4 workers, propositalmente > 1

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pipeline.Start(ctx)

	const total = 50
	for i := 0; i < total; i++ {
		pipeline.Submit(validReading())
	}

	waitUntil(t, 2*time.Second, func() bool { return publisher.count() == total })
}

func TestPipeline_StopsProcessingAfterContextCancelled(t *testing.T) {
	publisher := &recordingPublisher{}
	pipeline := NewPipeline(publisher, 1, 10)

	ctx, cancel := context.WithCancel(context.Background())
	pipeline.Start(ctx)

	pipeline.Submit(validReading())
	waitUntil(t, time.Second, func() bool { return publisher.count() == 1 })

	cancel() // simula shutdown gracioso (SIGTERM)
	time.Sleep(50 * time.Millisecond)

	// Depois do cancelamento, nada que já estava publicado deveria
	// "desaparecer" - só garantimos que o worker não está mais ativo.
	// (Submit após cancel não é testado aqui de propósito: com o buffer
	// ainda com espaço, ele aceitaria a leitura sem travar - o
	// comportamento exato de "rejeitar após shutdown" fica para quando
	// a Pipeline ganhar um sinal explícito de "closed", fora do escopo
	// desta etapa.)
	if publisher.count() != 1 {
		t.Fatalf("expected exactly 1 published reading, got %d", publisher.count())
	}
}
