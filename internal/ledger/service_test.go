package ledger

import(
	"testing"
	"context"
	"errors"
	"time"
	"github.com/google/uuid"
)
func TestCreateTransaction_MinEntries(t *testing.T) {
    
    // the table - a slice of test cases
    tests := []struct {
        name          string      // what this test case is called
        entryCount    int         // the input - how many entries to create
        expectedError error       // what error we expect back, nil means no error
    }{
        {
            name:          "one entry should fail",
            entryCount:    1,
            expectedError: ErrInsufficientEntries,
        },
        {
            name:          "two entries should pass validation",
            entryCount:    2,
            expectedError: nil, // nil means we don't expect this specific error
        },
        {
            name:          "three entries should pass validation",
            entryCount:    3,
            expectedError: nil,
        },
    }

    // the loop - runs the same logic against every case
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            
            // build a service with no database - validation runs before db calls
            svc := &Service{db: nil, pool: nil}

            // build a request with the right number of entries
            entries := make([]CreateEntryRequest, tt.entryCount)
            for i := range entries {
                entries[i] = CreateEntryRequest{
                    AmountCents: 100,
                    Direction:   "debit",
                    Currency:    "USD",
                }
            }

            req := CreateTransactionRequest{
                Description:    "test",
                IdempotencyKey: "test-key",
                OccurredAt:     time.Now(),
                Entries:        entries,
            }

            // call the function
            _, err := svc.CreateTransaction(context.Background(), req)

            // assert
            if tt.expectedError != nil {
                if !errors.Is(err, tt.expectedError) {
                    t.Errorf("test %s: expected error %v got %v", tt.name, tt.expectedError, err)
                }
            }
        })
    }
    }

//tests for balance
func TestCreateTransaction_Balance (t *testing.T){
tests := []struct {
        name          string      // what this test case is called
	entries []CreateEntryRequest
        expectedError error       // what error we expect back, nil means no error
    }{
        {
            name:          "unbalanced transaction should fail",
	    entries: []CreateEntryRequest{
		{AccountID: uuid.New(), AmountCents: 100, Direction: "debit", Currency: "USD"},
		{AccountID: uuid.New(), AmountCents: 50, Direction: "credit", Currency: "USD"},
	    },
            expectedError: ErrUnbalancedTransaction,
        },
                
    }

    // the loop - runs the same logic against every case
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            
            // build a service with no database - validation runs before db calls
            svc := &Service{db: nil, pool: nil}

            req := CreateTransactionRequest{
                Description:    "test",
                IdempotencyKey: "test-key",
                OccurredAt:     time.Now(),
                Entries:        tt.entries,
            }

            // call the function
            _, err := svc.CreateTransaction(context.Background(), req)

            // assert
            if tt.expectedError != nil {
                if !errors.Is(err, tt.expectedError) {
                    t.Errorf("test %s: expected error %v got %v", tt.name, tt.expectedError, err)
                }
            }
        })
}
}
    
