package ledger

import(
	"time"
	"github.com/google/uuid"
)
type CreateTransactionRequest struct{
	Description string
	IdempotencyKey string
	OccurredAt time.Time
	Entries []CreateEntryRequest
}

type CreateEntryRequest struct{
	AccountID uuid.UUID
	AmountCents int64
	Direction string
	Currency string
}
