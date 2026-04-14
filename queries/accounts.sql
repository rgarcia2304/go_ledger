-- name: CreateAccount :one
INSERT INTO accounts(name, currency, account_type)
VALUES($1, $2, $3)
RETURNING *;

-- name: GetAccount :one
SELECT * from accounts
WHERE id = $1 LIMIT 1; 

-- name: GetBalance :one
SELECT SUM(
	CASE WHEN direction = 'debit'
	THEN amount_cents
	ELSE -amount_cents
	END
) AS balance 
from entries
WHERE account_id = $1;
