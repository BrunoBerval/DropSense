# 🔷 farm-management (C# / .NET)

**Core Domain** do AgroFlow. É aqui que mora a maior parte do esforço de DDD/TDD do projeto — ver `docs/01-CONTEXTO-PROJETO.md` na raiz do repositório para o contexto completo da arquitetura.

## Status atual

- [x] Aggregate `Zone` — registro de zona + atualização de limites de SLA, com testes (TDD).
- [ ] Aggregate `Reservoir`
- [ ] Aggregate `IrrigationCycle`
- [ ] Aggregate `Alert`
- [ ] Application layer (handlers / use cases)
- [ ] Infrastructure (EF Core + Postgres, producer/consumer Kafka)
- [ ] API (ASP.NET Core, endpoints REST consumidos pelo React)

## Estrutura

```
src/
  FarmManagement.Domain/        ← lógica de negócio pura, zero dependência externa
    Zone.cs
    DomainEvents/
    Exceptions/
tests/
  FarmManagement.Domain.Tests/  ← xUnit
```

Por que `FarmManagement.Domain` não tem nenhum `PackageReference`? De propósito. O Domain layer em DDD não deveria depender de Kafka, Postgres ou ASP.NET — só de regras de negócio testáveis em isolamento. Isso é o que torna possível rodar a suíte de testes sem subir nenhuma infraestrutura.

## Aggregate: `Zone`

Representa uma zona de cultivo. Invariantes garantidas pelo próprio aggregate (não pela camada de aplicação, nem pelo banco):

| Invariante | Onde é validada |
|---|---|
| `ZoneId` não pode ser vazio | `Zone.Register` |
| `Hectares` deve ser > 0 | `Zone.Register` |
| Limites de SLA devem estar entre 0 e 100 | `Zone.UpdateSlaLimits` |
| `Min` deve ser estritamente menor que `Max` | `Zone.UpdateSlaLimits` |

Cada uma dessas regras tem um teste correspondente em `ZoneTests.cs` — esse é o contrato real do aggregate, mais confiável do que qualquer comentário.

### Domain Events gerados

- `ZoneRegistered` → publicado em `zone.events.v1`
- `ZoneSlaLimitsUpdated` → publicado em `zone.events.v1`

(A publicação de fato no Kafka ainda não existe — esses eventos hoje só vivem em memória, acessíveis via `zone.DomainEvents`. Isso vai para a Infrastructure layer.)

## Como rodar os testes

```bash
cd tests/FarmManagement.Domain.Tests
dotnet test
```

Esperado: **7 testes passando** (`Register` com dados válidos/inválidos, `UpdateSlaLimits` com limites válidos/inválidos).

## Decisões registradas

- **Por que `Zone` não guarda lista de leituras de sensor?** Isso pertenceria a outro aggregate (ou nem a aggregate nenhum — talvez só um read model), pra não inflar `Zone` com dados de alto volume que não fazem parte do seu invariante de negócio.
- **Por que `DomainValidationException` é uma exceção única, e não uma por invariante?** Pragmatismo de projeto educacional — o número de invariantes ainda é pequeno. Se crescer, vale revisitar.
