-- name: CreateEntry :one 
INSERT INTO entries(transaction_id, account_id, amount_cents, currency, direction)
VALUES($1, $2, $3, $4, $5)
RETURNING *; 

-- name: GetEntires :many
SELECT * from entries WHERE transaction_id = $1;

