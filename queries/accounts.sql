-- name: CreateAccount :one
INSERT INTO accounts(name, currency, account_type)
VALUES($1, $2, $3)
RETURNING *;

-- name: GetAccount :one
SELECT * from accounts
WHERE id = $1 LIMIT 1; 

-- name: GetBalance :one
SELECT 
COALESCE(
SUM(
	CASE WHEN direction = 'debit'
	THEN amount_cents
	ELSE -amount_cents
	END), 0
)::bigint AS balance 
from entries
WHERE account_id = $1;

-- name: GetAccountHistory :many 
SELECT transactions.description, entries.amount_cents, entries.direction, entries.created_at
from entries
INNER JOIN accounts
ON accounts.id = entries.account_id
INNER JOIN transactions
ON entries.transaction_id = transactions.id
WHERE entries.account_id = $1 
ORDER BY entries.created_at DESC; 
