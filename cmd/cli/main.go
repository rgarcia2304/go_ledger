package main 


import(
	"github.com/rgarcia2304/go_ledger/internal/ledger"
	"github.com/joho/godotenv"
	"log"
	db "github.com/rgarcia2304/go_ledger/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
	"os"
)


var svc *ledger.Service


func main(){
	err := godotenv.Load("../../.env")
	if err != nil{
		log.Fatalf("Could not load info from env variables: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil{
		log.Fatalf("Issue creating db pool: %w", err)
	}

	defer pool.Close()
	queries := db.New(pool)

	svc = ledger.NewService(queries, pool)

	Execute()

}
