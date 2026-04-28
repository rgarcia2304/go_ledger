package ledger 


import (
    "time"
    "database/sql"
    "log"
    "os"
    "testing"
    "context"
    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/pressly/goose/v3"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go"

    db "github.com/rgarcia2304/go_ledger/internal/db"
    "github.com/jackc/pgx/v5/pgxpool"
    "errors"
    "sync"
    "fmt"
    "github.com/google/uuid"
)

var testSvc *Service
var testContainer *postgres.PostgresContainer
var testPool *pgxpool.Pool

func TestMain(m *testing.M){
	
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, 
	"postgres:16-alpine",
	postgres.WithDatabase("ledger"),
	postgres.WithUsername("postgres"),
        postgres.WithPassword("ledger"),
        postgres.BasicWaitStrategies(),
        postgres.WithSQLDriver("pgx"),
    )

    	if err != nil{
		log.Fatalf("failed to start container: %v", err)
	}

	defer testcontainers.TerminateContainer(ctr)

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil{
		log.Fatalf("failed to get connection string: %v", err)
	}

	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil{
		log.Fatalf("failed to open db: %v", err)
	}


	if err := goose.SetDialect("postgres"); err != nil{
		log.Fatalf("failed to set dialect: %v", err)
	}

	if err := goose.Up(sqlDB, "../../migrations"); err != nil{
		log.Fatalf("failed to run migrations: %v", err)
	}
	
	
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil{
		log.Fatalf("error creating db pool: %v", err)
	}
	defer pool.Close()
	queries := db.New(pool)
	testSvc = NewService(queries, pool)

	testPool = pool

	code := m.Run()
	os.Exit(code)

}

func cleanDB(t *testing.T){
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		"TRUNCATE entries, transactions, accounts RESTART IDENTITY CASCADE")
	if err != nil{
		t.Fatalf("failed to clean database: %v", err)
	}
}

func TestCreateTransaction_IdempotencyRetry(t *testing.T) {
    cleanDB(t)	
    ctx := context.Background()

    // create two accounts for this test
    acc1, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company assets",
        AccountType: "asset",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 1: %v", err)
    }

    acc2, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company liability",
        AccountType: "liability",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 2: %v", err)
    }

    // build the request with a fixed idempotency key
    req := CreateTransactionRequest{
        Description:    "test idempotency",
        IdempotencyKey: "test-idempotency-key-123",
        OccurredAt:     time.Now(),
        Entries: []CreateEntryRequest{
            {AccountID: acc1.ID, AmountCents: 100, Direction: "debit", Currency: "USD"},
            {AccountID: acc2.ID, AmountCents: 100, Direction: "credit", Currency: "USD"},
        },
    }

    // first submission - should write and return new transaction
    result1, err := testSvc.CreateTransaction(ctx, req)
    if err != nil {
        t.Fatalf("first submission failed unexpectedly: %v", err)
    }

    // second submission with same key - should return existing transaction
    result2, err := testSvc.CreateTransaction(ctx, req)
    if err != nil {
        t.Fatalf("second submission failed unexpectedly: %v", err)
    }

    // both calls should return the same transaction ID
    if result1.ID != result2.ID {
        t.Errorf("expected same transaction ID got %v and %v", result1.ID, result2.ID)
    }
}  

func TestCreateTransaction_InsufficentBalance(t *testing.T) {
    cleanDB(t)	
    ctx := context.Background()

    // create two accounts for this test
    acc1, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company assets",
        AccountType: "asset",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 1: %v", err)
    }

    acc2, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company liability",
        AccountType: "liability",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 2: %v", err)
    }

    // build the request with a fixed idempotency key
    req := CreateTransactionRequest{
        Description:    "test idempotency",
        IdempotencyKey: "test-idempotency-key-123",
        OccurredAt:     time.Now(),
        Entries: []CreateEntryRequest{
            {AccountID: acc1.ID, AmountCents: 100, Direction: "debit", Currency: "USD"},
            {AccountID: acc2.ID, AmountCents: 100, Direction: "credit", Currency: "USD"},
        },
    }

    // first submission - should write and return new transaction
    result1, err := testSvc.CreateTransaction(ctx, req)
    if err != nil {
        t.Fatalf("first submission failed unexpectedly: %v", err)
    }

    // second submission with same key - should return existing transaction
    result2, err := testSvc.CreateTransaction(ctx, req)
    if err != nil {
        t.Fatalf("second submission failed unexpectedly: %v", err)
    }

    // both calls should return the same transaction ID
    if result1.ID != result2.ID {
        t.Errorf("expected same transaction ID got %v and %v", result1.ID, result2.ID)
    }

    //create second transaction 
    req = CreateTransactionRequest{
        Description:    "test idempotency_2",
        IdempotencyKey: "test-idempotency-key-1234",
        OccurredAt:     time.Now(),
        Entries: []CreateEntryRequest{
            {AccountID: acc1.ID, AmountCents: 500000, Direction: "credit", Currency: "USD"},
            {AccountID: acc2.ID, AmountCents: 500000, Direction: "debit", Currency: "USD"},
        },
    }

    result1, err = testSvc.CreateTransaction(ctx, req)
    if err != nil {
        if !errors.Is(err, ErrInsufficientFunds) {
                    t.Errorf("test %s: expected error %v got %v", "Insufficient Balance Test" , ErrInsufficientFunds, err)
                }
            }

    }

 
	 

func TestCreateTransaction_HappyPath(t *testing.T) {
    cleanDB(t)	
    ctx := context.Background()

    // create two accounts for this test
    acc1, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company assets",
        AccountType: "asset",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 1: %v", err)
    }

    acc2, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company liability",
        AccountType: "liability",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 2: %v", err)
    }

    // build the request with a fixed idempotency key
    req := CreateTransactionRequest{
        Description:    "test idempotency",
        IdempotencyKey: "test-idempotency-key-123",
        OccurredAt:     time.Now(),
        Entries: []CreateEntryRequest{
            {AccountID: acc1.ID, AmountCents: 100, Direction: "debit", Currency: "USD"},
            {AccountID: acc2.ID, AmountCents: 100, Direction: "credit", Currency: "USD"},
        },
    }

    // first submission - should write and return new transaction
    _, err = testSvc.CreateTransaction(ctx, req)
    if err != nil {
        t.Fatalf("first submission failed unexpectedly: %v", err)
    }

    accBalance, err := testSvc.GetBalance(ctx, acc1.ID)
    if err != nil{
	    t.Fatalf("Issue getting the accounts balance, unexpected account should exist: %v", err)
    }

    if accBalance != int64(100){
	t.Fatal("Account Balance Should Be 100")
    }

}

func TestCreateTransaction_ChaosTest(t *testing.T) {
    cleanDB(t)	
    ctx := context.Background()

    // create two accounts for this test
    acc1, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company assets",
        AccountType: "asset",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 1: %v", err)
    }

    if err != nil {
        t.Fatalf("failed to create account 2: %v", err)
    }

    //generate a random UUID 
    randomID:= uuid.New()

    // build the request with a fixed idempotency key
    req := CreateTransactionRequest{
        Description:    "test idempotency",
        IdempotencyKey: "test-idempotency-key-123",
        OccurredAt:     time.Now(),
        Entries: []CreateEntryRequest{
            {AccountID: acc1.ID, AmountCents: 100, Direction: "debit", Currency: "USD"},
            {AccountID: randomID, AmountCents: 100, Direction: "credit", Currency: "USD"},
        },
    }

    // first submission - should write and return new transaction
    _, err = testSvc.CreateTransaction(ctx, req)
    if err != nil {
        if !errors.Is(err, ErrAccountNotFound) {
                    t.Errorf("test %s: expected error %v got %v", "Insufficient Balance Test" , ErrAccountNotFound, err)
                }
        }
   
    history, err := testSvc.GetTransactionHistory(ctx, acc1.ID)
    if err != nil{
	t.Fatalf("failed to get history: %v", err)
    }
    
    if len(history) != 0 {
	t.Errorf("expected empty history but found %d entries", len(history))
    }

}

func TestCreateTransaction_ConcurrentWithdraw(t *testing.T) {
    cleanDB(t)	
    ctx := context.Background()

    // create two accounts for this test
    acc1, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company assets",
        AccountType: "asset",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 1: %v", err)
    }

    acc2, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company liability",
        AccountType: "liability",
        Currency:    "USD",
    })
    if err != nil {
        t.Fatalf("failed to create account 2: %v", err)
    }

    // build the request with a fixed idempotency key
    req := CreateTransactionRequest{
        Description:    "initial funding",
        IdempotencyKey: "fund-key-123",
        OccurredAt:     time.Now(),
        Entries: []CreateEntryRequest{
            {AccountID: acc1.ID, AmountCents: 10000, Direction: "debit", Currency: "USD"},
            {AccountID: acc2.ID, AmountCents: 10000, Direction: "credit", Currency: "USD"},
        },
    }



    // first submission - should write and return new transaction
    _, err = testSvc.CreateTransaction(ctx, req)
    if err != nil {
        t.Fatalf("funding failed unexpectedly: %v", err)
    }


    const numGoroutines = 10
    const withdrawAmount = 100
    errs := make(chan error, numGoroutines)

    var wg sync.WaitGroup
    for i := 0; i <10; i ++{
	wg.Add(1)
	i := i 
	go func(){
		defer wg.Done()
		req := CreateTransactionRequest{
                Description:    "withdrawal",
                IdempotencyKey: fmt.Sprintf("withdraw-key-%d", i),
                OccurredAt:     time.Now(),
                Entries: []CreateEntryRequest{
                    {AccountID: acc1.ID, AmountCents: withdrawAmount, Direction: "credit", Currency: "USD"},
                    {AccountID: acc2.ID, AmountCents: withdrawAmount, Direction: "debit", Currency: "USD"},
                },
            }
		_, err := testSvc.CreateTransaction(ctx, req)
		if err != nil{
			errs <- err
		}
	
	}()
    }

    wg.Wait()
    close(errs)

    for err := range errs {
	t.Errorf("go routine failed: %v", err)
    }

    accBalance, err := testSvc.GetBalance(ctx, acc1.ID)
    if err != nil{
	    t.Fatalf("Issue getting the accounts balance, unexpected account should exist: %v", err)
    }

    expected := int64(10000 - (numGoroutines * withdrawAmount))
    if accBalance != expected{
	    t.Fatalf("Account Balance Should Be: %v, curr balance is %v", expected, accBalance)
    }

}

 
	 
func BenchmarkCreateTransaction(b *testing.B) {
    ctx := context.Background()
    // create two accounts for this test
    acc1, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company assets",
        AccountType: "asset",
        Currency:    "USD",
    })
    if err != nil {
        b.Fatalf("failed to create account 1: %v", err)
    }

    acc2, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company liability",
        AccountType: "liability",
        Currency:    "USD",
    })
    if err != nil {
        b.Fatalf("failed to create account 2: %v", err)
    }

    b.ResetTimer() 

    for i := 0; i < b.N; i++ {
        // the thing you're measuring - runs b.N times

	req := CreateTransactionRequest{
        Description: "benchmark transaction",
        IdempotencyKey: uuid.New().String(),
        OccurredAt:     time.Now(),
        Entries: []CreateEntryRequest{
            {AccountID: acc1.ID, AmountCents: 10000, Direction: "debit", Currency: "USD"},
            {AccountID: acc2.ID, AmountCents: 10000, Direction: "credit", Currency: "USD"},
        },
    }



    // first submission - should write and return new transaction
    _, err = testSvc.CreateTransaction(ctx, req)
    if err != nil {
        b.Fatalf("funding failed unexpectedly: %v", err)

    }

    }
}

func BenchmarkCreateTransaction_Parallel(b *testing.B) {

    ctx := context.Background()	
    acc1, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company assets",
        AccountType: "asset",
        Currency:    "USD",
    })
    if err != nil {
        b.Fatalf("failed to create account 1: %v", err)
    }

    acc2, err := testSvc.CreateAccount(ctx, CreateAccountRequest{
        Name:        "company liability",
        AccountType: "liability",
        Currency:    "USD",
    })
    if err != nil {
        b.Fatalf("failed to create account 1: %v", err)
    }
    b.ResetTimer()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req := CreateTransactionRequest{
                IdempotencyKey: uuid.New().String(),
		OccurredAt: time.Now(),
		Entries: []CreateEntryRequest{
            {AccountID: acc1.ID, AmountCents: 10000, Direction: "debit", Currency: "USD"},
            {AccountID: acc2.ID, AmountCents: 10000, Direction: "credit", Currency: "USD"},
        	},
            }
            _, err := testSvc.CreateTransaction(
                context.Background(), req)
            if err != nil {
                b.Errorf("transaction failed: %v", err)
            }
        }
    })
}
 
	 

 
