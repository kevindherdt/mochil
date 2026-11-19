package metadata

import (
	"errors"
	"testing"
)

func TestTransactionNeedsAttentionRecognizesWrappedFailure(t *testing.T) {
	err := &TransactionError{Err: errors.New("rollback failed")}
	if !TransactionNeedsAttention(err) || TransactionNeedsAttention(errors.New("apply rejected")) {
		t.Fatal("transaction error classification is incorrect")
	}
}
