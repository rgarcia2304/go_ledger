package ledger 
import(

)

type CreateTransactionRequest struct{
	Description string
	IdempotencyKey string
	OccuredAt time.Time
	Entries []CreateEntryRequest
}

type CreateEntryRequest(
	AccountID uuid.UUID
	AmountCents int64
	Direction string
	Currency string
)


func(s *Service) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*db.Transaction, error){

}
