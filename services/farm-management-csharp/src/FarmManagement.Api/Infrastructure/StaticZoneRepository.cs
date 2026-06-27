using FarmManagement.Api.Application;
using FarmManagement.Api.Domain;

namespace FarmManagement.Api.Infrastructure;

// PROVISÓRIO, por decisão explícita do usuário: o evento ZoneRegistered
// (que deveria ser a fonte real desta lista) é publicado por este
// mesmo serviço, via uma API REST que ainda não existe (não há
// frontend consumindo nada ainda). Até essa fonte real existir, a
// lista de zonas é estática - mesmo papel de weather.Zones() no Go.
//
// zone-042 não está em manutenção ("por enquanto tudo funcionando") e
// o reservatório sempre tem volume suficiente (Reservoir() usa o
// default true) - exatamente como combinado.
public sealed class StaticZoneRepository : IZoneRepository
{
    private readonly Dictionary<string, Zone> _zones = new()
    {
        ["zone-042"] = new Zone("zone-042", isUnderMaintenance: false, new Reservoir()),
    };

    public Zone? Find(string zoneId) => _zones.TryGetValue(zoneId, out var zone) ? zone : null;
}
