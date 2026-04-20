package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/google/uuid"
    "github.com/rgarcia2304/go_ledger/internal/ledger"
    "github.com/spf13/cobra"
    "crypto/sha256"
    "encoding/hex"
)

var transactionFile string

type EntryReq struct {
    AccountID   uuid.UUID `json:"account_id"`
    AmountCents int64     `json:"amount_cents"`
    Direction   string    `json:"direction"`
    Currency    string    `json:"currency"`
}

type TransactionJson struct {
    Description string     `json:"description"`
    OccurredAt  time.Time  `json:"occurred_at"`
    Entries     []EntryReq `json:"entries"`
}
var getTransactionHistoryCmd = &cobra.Command{
	Use: "get-transaction-history",
	Aliases: []string{"gth"},
	Short: "Gets the history of transactions on ledger",
	RunE: func(cmd *cobra.Command, args []string)error{
		parsedID, err := uuid.Parse(accountID)
		if err != nil{
			return err
		}
		transactions, err := svc.GetTransactionHistory(context.Background(), parsedID)
		if err != nil{
			return err
		}
		fmt.Printf("Transaction History for Account ID %v", accountID)
		for _, t := range(transactions){
			fmt.Printf("Transaction: %v", t)
		}
		return nil
	},
}
var createTransactionCmd = &cobra.Command{
    Use:     "create-transaction",
    Aliases: []string{"ct"},
    Short:   "Creates a new ledger transaction",
    RunE: func(cmd *cobra.Command, args []string) error {

        // read the json file
        data, err := os.ReadFile(transactionFile)
        if err != nil {
            return fmt.Errorf("could not read file: %w", err)
        }

        // unmarshal into struct
        var txFile TransactionJson
        if err := json.Unmarshal(data, &txFile); err != nil {
            return fmt.Errorf("could not parse transaction file: %w", err)
        }

        // validate document completeness
        if txFile.Description == "" {
            return fmt.Errorf("description is required")
        }
        if txFile.OccurredAt.IsZero() {
            return fmt.Errorf("occurred_at is required")
        }
        if len(txFile.Entries) == 0 {
            return fmt.Errorf("at least one entry is required")
        }
        for i, entry := range txFile.Entries {
            if entry.AccountID == uuid.Nil {
                return fmt.Errorf("entry %d: account_id is required", i)
            }
            if entry.AmountCents <= 0 {
                return fmt.Errorf("entry %d: amount_cents must be greater than zero", i)
            }
            if entry.Direction != "debit" && entry.Direction != "credit" {
                return fmt.Errorf("entry %d: direction must be debit or credit", i)
            }
            if entry.Currency == "" {
                return fmt.Errorf("entry %d: currency is required", i)
            }
        }

        // generate idempotency key
	hash := sha256.Sum256(data)
	idempotencyKey := hex.EncodeToString(hash[:])

        // build entries list
        var entriesReqList []ledger.CreateEntryRequest
        for _, entry := range txFile.Entries {
            entriesReqList = append(entriesReqList, ledger.CreateEntryRequest{
                AccountID:   entry.AccountID,
                AmountCents: entry.AmountCents,
                Direction:   entry.Direction,
                Currency:    entry.Currency,
            })
        }

        // build transaction request
        transactionRequest := ledger.CreateTransactionRequest{
            Description:    txFile.Description,
            IdempotencyKey: idempotencyKey,
            OccurredAt:     txFile.OccurredAt,
            Entries:        entriesReqList,
        }

        // submit to service
        ctx := context.Background()
        result, err := svc.CreateTransaction(ctx, transactionRequest)
        if err != nil {
            return err
        }

        fmt.Printf("Transaction recorded successfully\nID: %s\n", result.ID)
        return nil
    },
}

func init() {
    createTransactionCmd.Flags().StringVar(&transactionFile, "file", "", "Path to transaction JSON file (required)")
    getTransactionHistoryCmd.Flags().StringVar(&accountID, "accountID", "", "Account ID (required)")
    getTransactionHistoryCmd.MarkFlagRequired("accountID")
    rootCmd.AddCommand(getTransactionHistoryCmd)
    createTransactionCmd.MarkFlagRequired("file")
    rootCmd.AddCommand(createTransactionCmd)
}
