package persistence

// OutboxTable is this service's outbox; rows are written through the shared
// libs/common/outbox package in the same transaction as the business change.
const OutboxTable = "user_service.outbox_events"
