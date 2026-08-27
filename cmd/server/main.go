package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	db "github.com/rgarcia2304/go_ledger/internal/db"
	"github.com/rgarcia2304/go_ledger/internal/handler"
	"github.com/rgarcia2304/go_ledger/internal/ledger"
)

func main() {
	// load env
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// create connection pool
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("failed to parse db config: %v", err)
	}
	config.MaxConns = 25

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	// verify connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Println("connected to database")

	// wire up layers
	queries := db.New(pool)
	svc := ledger.NewService(queries, pool)
	h := handler.NewHandler(svc)

	// register routes
	mux := http.NewServeMux()

	mux.HandleFunc("POST /accounts", h.CreateAccount)
	mux.HandleFunc("POST /transactions", h.CreateTransaction)
	mux.HandleFunc("GET /accounts/{id}/balance", h.GetBalance)
	mux.HandleFunc("GET /accounts/{id}/history", h.GetTransactionHistory)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
    	http.ServeFile(w, r, "static/demo.html")
	})

	mux.HandleFunc("GET /api", func(w http.ResponseWriter, r *http.Request) {
    	http.ServeFile(w, r, "static/index.html")
	})

	// start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
