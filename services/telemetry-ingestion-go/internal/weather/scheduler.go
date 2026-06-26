package weather

import (
	"context"
	"log"
	"time"
)

// forecastWindowHours é quantas horas adiante o forecast cobre -
// "previsão de chuva para as próximas X horas", como o README descreve
// na Parte 2, e bate com o exemplo de WeatherForecastUpdated
// (forecastWindowHours: 12).
const forecastWindowHours = 12

// ForecastFetcher é o que o Scheduler precisa para buscar previsão de
// uma zona. Hoje só *OpenMeteoClient implementa; existir como
// interface permite testar o Scheduler com um fetcher fake, sem rede.
type ForecastFetcher interface {
	FetchForecast(ctx context.Context, zone Zone, windowHours int) (Forecast, error)
}

// ForecastPublisher é o que o Scheduler precisa para publicar. Hoje
// implementado por *kafka.Producer, mas declarado AQUI (não em
// internal/kafka) - mesmo raciocínio de Submitter em handler.go e
// Publisher em pipeline.go: quem consome a interface é quem a
// declara, não quem a implementa. Assim este pacote não precisa
// importar internal/kafka.
type ForecastPublisher interface {
	PublishWeatherForecast(ctx context.Context, zoneID string, forecast Forecast) error
}

// Scheduler busca e publica previsão do tempo para cada zona
// monitorada, em intervalos regulares - "em intervalos regulares,
// busca a probabilidade de chuva", conforme o README descreve esse
// serviço fazendo, na Parte 2.
type Scheduler struct {
	fetcher   ForecastFetcher
	publisher ForecastPublisher
	zones     []Zone
	interval  time.Duration
}

// NewScheduler monta o scheduler. zones é a lista consultada a cada
// ciclo - hoje vem de Zones() (estática); ver o comentário em zone.go
// sobre isso ser provisório.
func NewScheduler(fetcher ForecastFetcher, publisher ForecastPublisher, zones []Zone, interval time.Duration) *Scheduler {
	return &Scheduler{
		fetcher:   fetcher,
		publisher: publisher,
		zones:     zones,
		interval:  interval,
	}
}

// Start dispara um ciclo imediatamente (não espera o primeiro tick do
// Ticker) e repete a cada interval, até ctx ser cancelado. Mesmo
// padrão de "roda em goroutine própria, para com ctx" do
// Pipeline.worker em pipeline.go.
func (s *Scheduler) Start(ctx context.Context) {
	go s.run(ctx)
}

func (s *Scheduler) run(ctx context.Context) {
	s.tick(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// tick processa cada zona de forma independente: uma zona falhando ao
// buscar ou publicar não impede as outras de seguirem - mesma filosofia
// de "loga e segue" que Pipeline.worker já usa para erro de publish
// (dead-letter é um TODO documentado lá, fora de escopo aqui também).
func (s *Scheduler) tick(ctx context.Context) {
	for _, zone := range s.zones {
		forecast, err := s.fetcher.FetchForecast(ctx, zone, forecastWindowHours)
		if err != nil {
			log.Printf("failed to fetch forecast for zone %s: %v", zone.ID, err)
			continue
		}
		if err := s.publisher.PublishWeatherForecast(ctx, zone.ID, forecast); err != nil {
			log.Printf("failed to publish forecast for zone %s: %v", zone.ID, err)
		}
	}
}
