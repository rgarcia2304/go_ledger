-- name: CreateTransaction :one
INSERT INTO transactions(description, idempotency_key, occurred_at)
VALUES($1, $2, $3)
RETURNING *; 

-- name: GetTransactionByIdempotencyKey :one
SELECT * from transactions WHERE idempotency_key  = $1;

