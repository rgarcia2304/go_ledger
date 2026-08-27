package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rgarcia2304/go_ledger/internal/ledger"
	"github.com/google/uuid"
	"time"
	"fmt"
)

type Handler struct {
	svc *ledger.Service
}

func NewHandler(svc *ledger.Service) *Handler {
	return &Handler{svc: svc}
}


func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// POST /accounts

type CreateAccountRequest struct {
	Name        string `json:"name"`
	AccountType string `json:"type"`
	Currency    string `json:"currency"`
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.AccountType == "" || req.Currency == "" {
		writeError(w, http.StatusBadRequest, "name, type, and currency are required")
		return
	}

	validTypes := map[string]bool{
		"asset": true, "liability": true, "equity": true,
		"revenue": true, "expense": true,
	}
	if !validTypes[req.AccountType] {
		writeError(w, http.StatusBadRequest, "type must be asset, liability, equity, revenue, or expense")
		return
	}

	acc, err := h.svc.CreateAccount(r.Context(), ledger.CreateAccountRequest{
		Name:        req.Name,
		AccountType: req.AccountType,
		Currency:    req.Currency,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	writeJSON(w, http.StatusCreated, acc)
}

// POST /transactions

type CreateTransactionRequest struct {
	Description    string               `json:"description"`
	OccurredAt     string               `json:"occurred_at"`
	IdempotencyKey string               `json:"idempotency_key"`
	Entries        []CreateEntryRequest `json:"entries"`
}

type CreateEntryRequest struct {
	AccountID   uuid.UUID `json:"account_id"`
	AmountCents int64     `json:"amount_cents"`
	Direction   string    `json:"direction"`
	Currency    string    `json:"currency"`
}

func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}
	if req.IdempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key is required")
		return
	}
	if len(req.Entries) < 2 {
		writeError(w, http.StatusBadRequest, "at least two entries are required")
		return
	}

	// parse occurred_at
	occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "occurred_at must be RFC3339 format e.g. 2026-04-16T10:00:00Z")
		return
	}

	// validate entries
	for i, entry := range req.Entries {
		if entry.AccountID == uuid.Nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("entry %d: account_id is required", i))
			return
		}
		if entry.AmountCents <= 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("entry %d: amount_cents must be greater than zero", i))
			return
		}
		if entry.Direction != "debit" && entry.Direction != "credit" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("entry %d: direction must be debit or credit", i))
			return
		}
		if entry.Currency == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("entry %d: currency is required", i))
			return
		}
	}

	// build service request
	entries := make([]ledger.CreateEntryRequest, len(req.Entries))
	for i, e := range req.Entries {
		entries[i] = ledger.CreateEntryRequest{
			AccountID:   e.AccountID,
			AmountCents: e.AmountCents,
			Direction:   e.Direction,
			Currency:    e.Currency,
		}
	}

	tx, err := h.svc.CreateTransaction(r.Context(), ledger.CreateTransactionRequest{
		Description:    req.Description,
		IdempotencyKey: req.IdempotencyKey,
		OccurredAt:     occurredAt,
		Entries:        entries,
	})
	if err != nil {
		switch {
		case errors.Is(err, ledger.ErrInsufficientFunds):
			writeError(w, http.StatusUnprocessableEntity, "insufficient funds")
		case errors.Is(err, ledger.ErrAccountNotFound):
			writeError(w, http.StatusBadRequest, "one or more accounts not found")
		case errors.Is(err, ledger.ErrUnbalancedTransaction):
			writeError(w, http.StatusBadRequest, "transaction entries do not balance")
		default:
			writeError(w, http.StatusInternalServerError, "failed to process transaction")
		}
		return
	}

	writeJSON(w, http.StatusCreated, tx)
}

// GET /accounts/{id}/balance

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	balance, err := h.svc.GetBalance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get balance")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"balance_cents": balance})
}

// GET /accounts/{id}/history

func (h *Handler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	history, err := h.svc.GetTransactionHistory(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get transaction history")
		return
	}

	writeJSON(w, http.StatusOK, history)
}
