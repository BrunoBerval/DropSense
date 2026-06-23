namespace FarmManagement.Domain.DomainEvents;

/// <summary>
/// Marker interface for every domain event raised by an aggregate.
/// These are the "facts" that get translated into Kafka events at the
/// infrastructure layer (see FarmManagement.Infrastructure - not built yet).
/// </summary>
public interface IDomainEvent
{
    Guid EventId { get; }
    DateTime OccurredAt { get; }
}
