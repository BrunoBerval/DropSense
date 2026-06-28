package sender

import (
	"context"
	"log"
	"sync"
	"time"

	"dropsense/mock-sensors/internal/csvsource"
)

// Sender é o que o Runner precisa para mandar uma leitura - hoje
// implementado por *Client, mas declarado aqui (não em client.go)
// pelo mesmo motivo de ForecastPublisher no weather.Scheduler (Go) e
// IIrrigationEventPublisher (C#): quem consome a interface é quem a
// declara, não quem a implementa.
type Sender interface {
	Send(ctx context.Context, reading csvsource.Reading) error
}

// Runner percorre os ticks do CSV, disparando todos os sensores de
// um tick em paralelo a cada interval de tempo REAL - deliberadamente
// independente dos 30s simulados embutidos no CSV (ver csvsource).
// Quando loop=true, reinicia do primeiro tick ao chegar no fim, em
// vez de parar.
type Runner struct {
	sender   Sender
	ticks    [][]csvsource.Reading
	interval time.Duration
	loop     bool
}

func NewRunner(sender Sender, ticks [][]csvsource.Reading, interval time.Duration, loop bool) *Runner {
	return &Runner{sender: sender, ticks: ticks, interval: interval, loop: loop}
}

// Run dispara o primeiro tick imediatamente - mesmo padrão de
// "dispara já, espera depois" do weather.Scheduler e do
// DecisionScheduler - e segue até ctx ser cancelado ou (se loop for
// false) até o CSV terminar.
func (r *Runner) Run(ctx context.Context) {
	if len(r.ticks) == 0 {
		log.Println("mock-sensors: CSV sem nenhuma leitura, nada a fazer")
		return
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	index := 0
	for {
		r.sendTick(ctx, r.ticks[index])

		index++
		if index >= len(r.ticks) {
			if !r.loop {
				log.Println("mock-sensors: fim do CSV, loop desligado, encerrando")
				return
			}
			log.Println("mock-sensors: fim do CSV, reiniciando do primeiro tick (loop)")
			index = 0
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// sendTick dispara todos os sensores desse instante em paralelo - é,
// literalmente, várias centenas de sensores físicos reportando perto
// do mesmo momento, não uma fila artificial. Um sensor falhando não
// afeta os outros - mesma filosofia de "loga e segue" do
// Pipeline.worker no telemetry-ingestion.
func (r *Runner) sendTick(ctx context.Context, readings []csvsource.Reading) {
	var wg sync.WaitGroup
	for _, reading := range readings {
		wg.Add(1)
		go func(rd csvsource.Reading) {
			defer wg.Done()
			if err := r.sender.Send(ctx, rd); err != nil {
				log.Printf("mock-sensors: failed to send reading from sensor %s: %v", rd.SensorID, err)
			}
		}(reading)
	}
	wg.Wait()
}
