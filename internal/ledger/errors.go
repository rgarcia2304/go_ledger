package ledger 

import "errors"

var(

	ErrInsufficientEntries = errors.New("Transaction must have at least two entries")
	ErrUnbalancedTransaction = errors.New("Transaction entries do not balance: debits must equal credits")
	ErrAccountNotFound = errors.New("Entry references an account that do not exist")
	ErrInsufficientFunds = errors.New("insufficient funds: debit would exceed account balance")
	ErrMaxRetriesExceeded = errors.New("transaction failed after maximum retry attempts")

)
