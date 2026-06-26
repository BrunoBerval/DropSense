package weather

// Zone é uma área de cultivo com coordenadas, usada para consultar a
// previsão do tempo.
//
// PROVISÓRIO: hoje é uma lista estática porque o evento ZoneRegistered
// (que deveria ser a fonte real dessa informação) é publicado pelo
// Farm Management em C#, que ainda não existe. Quando existir, isso
// troca para um consumidor de zone.events.v1 - ver o Context Map no
// README ("Cs -->|Zone Events| Go").
type Zone struct {
	ID        string
	Latitude  float64
	Longitude float64
}

// Zones lista as zonas monitoradas hoje. zone-042 reaproveita o id de
// exemplo usado em todo o resto do projeto (testes, README); as
// coordenadas são de Três Pontas, MG - tradicionalmente conhecida como
// uma das capitais do café no Brasil, coerente com a "Fazenda Coador
// de Pano" descrita no README.
func Zones() []Zone {
	return []Zone{
		{ID: "zone-042", Latitude: -21.3700, Longitude: -45.2908},
	}
}
