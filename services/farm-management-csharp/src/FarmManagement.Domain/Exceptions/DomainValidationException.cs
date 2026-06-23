namespace FarmManagement.Domain.Exceptions;

/// <summary>
/// Thrown when an operation would break a domain invariant
/// (e.g. negative hectares, an SLA range that doesn't make sense).
/// Kept as a single exception type on purpose, for now: the project
/// is still small enough that splitting into many specific exception
/// classes would add ceremony without adding clarity. Revisit if the
/// number of distinct invariants grows.
/// </summary>
public sealed class DomainValidationException : Exception
{
    public DomainValidationException(string message) : base(message)
    {
    }
}
