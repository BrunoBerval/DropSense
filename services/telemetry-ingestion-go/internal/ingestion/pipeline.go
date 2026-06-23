package ingestion

import (
	"context"
	"log"
)

// Pipeline absorbs already-validated readings through a buffered
// channel and distributes them across a fixed pool of worker
// goroutines, each one forwarding to a Publisher.
//
// Why this exists: the HTTP handler validates synchronously (cheap,
// CPU-only - see reading.go) but must NOT call Publish synchronously,
// because publishing is the slow, I/O-bound part (network call to
// Kafka). A single slow publish must never stall the next incoming
// sensor request. The channel decouples "accepting a reading" from
// "doing something with it", and `workers` goroutines drain it in
// parallel so the work actually gets done concurrently, not just queued.
type Pipeline struct {
	readings  chan SoilReading
	publisher Publisher
	workers   int
}

// NewPipeline creates a pipeline with the given number of worker
// goroutines and channel buffer size.
//
//   - workers controls how much publishing happens concurrently.
//   - bufferSize controls how many readings can wait in the channel
//     before Submit starts blocking the caller - this is the
//     backpressure valve: once full, the HTTP layer feels the
//     slowdown instead of memory growing unbounded.
func NewPipeline(publisher Publisher, workers, bufferSize int) *Pipeline {
	return &Pipeline{
		readings:  make(chan SoilReading, bufferSize),
		publisher: publisher,
		workers:   workers,
	}
}

// Start launches the worker goroutines. Call once, typically from
// main(). Workers run until ctx is cancelled.
func (p *Pipeline) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		go p.worker(ctx)
	}
}

func (p *Pipeline) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case reading, ok := <-p.readings:
			if !ok {
				return
			}
			if err := p.publisher.Publish(ctx, reading); err != nil {
				log.Printf("failed to publish reading from sensor %s: %v", reading.SensorID, err)
				// TODO: estratégia de retry/dead-letter - fora do
				// escopo desta etapa. Por ora, erro de publish só é
				// logado; a leitura é perdida.
			}
		}
	}
}

// Submit hands an already-validated reading to the pipeline. It's the
// only method the HTTP layer touches. Non-blocking while the buffer
// has room; once full, it blocks the caller - deliberate backpressure
// instead of an unbounded queue.
func (p *Pipeline) Submit(reading SoilReading) {
	p.readings <- reading
}
