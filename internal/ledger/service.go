package ledger

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"fmt"

	db "github.com/rgarcia2304/go_ledger/internal/db"
)

type Service struct {
	db   *db.Queries
	pool *pgxpool.Pool
}

func NewService(queries *db.Queries, pool *pgxpool.Pool) *Service {
	return &Service{db: queries, pool: pool}
}

func (s *Service) CreateAccount(ctx context.Context, req CreateAccountRequest) (*db.Account, error){
	
	//ok lets looks at the reqs to create an account the entirety of the struct has to be fullfilled 

	acc, err := s.db.CreateAccount(ctx, db.CreateAccountParams{
		AccountType: req.AccountType, 
		Currency: req.Currency, 
		Name: req.Name, 
	})

	if err != nil{
		return nil, fmt.Errorf("Account could not be created: %w", err)
	}
	return &acc, nil
	
}

func (s *Service) GetBalance(ctx context.Context, accID uuid.UUID) (int64, error){
	
	_, err := s.db.GetAccount(ctx, accID)
	if err != nil{
		return int64(0), fmt.Errorf("could not create account: %w", err)
	}
	accountBalance, err := s.db.GetBalance(ctx, accID)
	if err != nil{
		return int64(0), fmt.Errorf("Account Could Not Be Found: %w", err)
	}

	return accountBalance, nil
}

func (s *Service) GetTransactionHistory(ctx context.Context, accID uuid.UUID) ([]db.GetAccountHistoryRow, error){
	
        _, err := s.db.GetAccount(ctx, accID)
	if err != nil{
		return nil, fmt.Errorf("could not create account: %w", err)
	}

	accHistory, err := s.db.GetAccountHistory(ctx, accID)
	if err != nil{
		return nil, fmt.Errorf("could not get history for this account: %w", err)
	}
	return accHistory, nil
}	
func (s *Service) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*db.Transaction, error) {

	// step 1 - validate min entries
	if len(req.Entries) < 2 {
		return nil, ErrInsufficientEntries
	}

	// step 2 - validate entries balance
	balance := int64(0)
	for _, entry := range req.Entries {
		if entry.Direction == "credit" {
			balance -= entry.AmountCents
		} else {
			balance += entry.AmountCents
		}
	}
	if balance != 0 {
		return nil, ErrUnbalancedTransaction
	}

	// step 3 - validate accounts exist and build asset credit map
	// only track asset accounts since those are the only ones
	// that can run out of funds
	assetsMap := make(map[uuid.UUID]int64)
	for _, entry := range req.Entries {
		acc, err := s.db.GetAccount(ctx, entry.AccountID)
		if err != nil {
			return nil, ErrAccountNotFound
		}
		if acc.AccountType == "asset" {
			assetsMap[acc.ID] = 0
		}
	}

	// check idempotency key before entering retry loop
	// handles the common case of a caller retrying after a timeout
	existing, err := s.db.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
	if err == nil {
		// transaction already exists, return it
		return &existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		// real database error, not just missing row
		return nil, err
	}

	// retry loop wrapping the serializable transaction
	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {

		result, err := s.attemptCreateTransaction(ctx, req, assetsMap)
		if err == nil {
			return result, nil
		}

		if isSerializationFailure(err) {
			// PostgreSQL detected a conflict, retry the whole attempt
			continue
		}

		if isIdempotencyDuplicate(err) {
			// concurrent request just wrote this transaction
			// fetch and return it
			existing, fetchErr := s.db.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return &existing, nil
		}

		// real error, do not retry
		return nil, err
	}

	return nil, ErrMaxRetriesExceeded
}

// attemptCreateTransaction runs the serializable transaction attempt.
// Extracted into its own function so defer tx.Rollback fires correctly
// at the end of each attempt rather than at the end of CreateTransaction.
func (s *Service) attemptCreateTransaction(ctx context.Context, req CreateTransactionRequest, assetsMap map[uuid.UUID]int64) (*db.Transaction, error) {

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) // safe no-op if commit succeeds

	qtx := s.db.WithTx(tx)

	// accumulate credits against asset accounts only
	// we need to know the total credits hitting each asset account
	// so we can check the account has sufficient funds to cover them
	creditTotals := make(map[uuid.UUID]int64)
	for _, entry := range req.Entries {
		if _, ok := assetsMap[entry.AccountID]; ok {
			if entry.Direction == "credit" {
				creditTotals[entry.AccountID] += entry.AmountCents
			}
		}
	}

	// check sufficient funds for each asset account being credited
	for accID, credits := range creditTotals {
		bal, err := qtx.GetBalance(ctx, accID)
		if err != nil {
			return nil, err
		}
		if bal - credits < 0 {
			return nil, ErrInsufficientFunds
		}
	}

	// write transaction row first - entries reference this via foreign key
	transRes, err := qtx.CreateTransaction(ctx, db.CreateTransactionParams{
		Description:    req.Description,
		IdempotencyKey: req.IdempotencyKey,
		OccurredAt:     req.OccurredAt,
	})
	if err != nil {
		return nil, err
	}

	// write entry rows - each entry points back to the transaction
	for _, entry := range req.Entries {
		_, err := qtx.CreateEntry(ctx, db.CreateEntryParams{
			TransactionID: transRes.ID,
			AccountID:     entry.AccountID,
			AmountCents:   entry.AmountCents,
			Direction:     entry.Direction,
			Currency:      entry.Currency,
		})
		if err != nil {
			return nil, err
		}
	}

	// commit - if this returns a serialization failure the caller retries
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &transRes, nil
}

func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}

func isIdempotencyDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" &&
		pgErr.ConstraintName == "transactions_idempotency_key_key"
}
