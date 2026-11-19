package metadata

import "errors"

// TransactionError marks a failed compensating action. Callers should surface
// it as an operational error because canonical state may require attention.
type TransactionError struct{ Err error }

func (e *TransactionError) Error() string { return e.Err.Error() }
func (e *TransactionError) Unwrap() error { return e.Err }

func TransactionNeedsAttention(err error) bool {
	var transaction *TransactionError
	return errors.As(err, &transaction)
}
