package metadata

import (
	"errors"
	"fmt"

	"mochil/internal/catalog"
)

type CommitRequest struct {
	JobID           string
	LibraryID       string
	OverrideChanges map[string]map[string]string
	RollbackSource  func() error
}

// CommitReview is the outer transaction shared by every writable metadata
// adapter. Source mutations have already succeeded when it is called. It adds
// optional catalogue overrides, records the durable applied state, and restores
// both layers if that final commit cannot be completed.
func CommitReview(jobs *JobStore, overrides *catalog.OverrideStore, request CommitRequest) (Job, error) {
	if jobs == nil {
		return Job{}, errors.New("metadata review storage is unavailable")
	}
	rollbackSource := request.RollbackSource
	if rollbackSource == nil {
		rollbackSource = func() error { return nil }
	}
	undoOverrides := func() error { return nil }
	if len(request.OverrideChanges) != 0 {
		if overrides == nil {
			if rollbackErr := rollbackSource(); rollbackErr != nil {
				return Job{}, &TransactionError{Err: fmt.Errorf("catalogue override storage is unavailable and source rollback needs attention: %w", rollbackErr)}
			}
			return Job{}, errors.New("catalogue override storage is unavailable; source changes were restored")
		}
		var err error
		undoOverrides, err = overrides.ApplyWithUndo(request.LibraryID, request.OverrideChanges)
		if err != nil {
			if rollbackErr := rollbackSource(); rollbackErr != nil {
				return Job{}, &TransactionError{Err: fmt.Errorf("could not store catalogue overrides and source rollback needs attention: %w", rollbackErr)}
			}
			return Job{}, fmt.Errorf("could not store catalogue overrides; source changes were restored: %w", err)
		}
	}
	updated, err := jobs.MarkApplied(request.JobID)
	if err == nil {
		return updated, nil
	}
	overrideRollbackErr := undoOverrides()
	sourceRollbackErr := rollbackSource()
	if sourceRollbackErr != nil || overrideRollbackErr != nil {
		return Job{}, &TransactionError{Err: errors.Join(
			fmt.Errorf("could not record metadata apply: %w", err),
			wrapCommitRollback("source rollback", sourceRollbackErr),
			wrapCommitRollback("catalogue override rollback", overrideRollbackErr),
		)}
	}
	return Job{}, fmt.Errorf("could not record metadata apply; all changes were restored: %w", err)
}

func wrapCommitRollback(label string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", label, err)
}
