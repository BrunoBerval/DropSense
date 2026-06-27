namespace FarmManagement.Api.Domain;

// HasSufficientVolume existe como conceito de domínio real - é uma
// das duas perguntas que o README define para este Core Domain
// responder antes de iniciar uma irrigação. Mas, por decisão
// explícita para esta fase do projeto (estudo de arquitetura de
// eventos, não de agronomia/operação de reservatório), o valor é
// sempre true: "o reservatório sempre deve ter volume". O dia que
// isso precisar refletir um sensor de nível real, só esta classe
// muda - Zone e DecisionHandler não sabem nem precisam saber de onde
// o valor vem.
public sealed class Reservoir
{
    public bool HasSufficientVolume { get; }

    public Reservoir(bool hasSufficientVolume = true)
    {
        HasSufficientVolume = hasSufficientVolume;
    }
}
