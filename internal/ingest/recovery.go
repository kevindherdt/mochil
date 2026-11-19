package ingest

type RecoveryAction string

const (
	RecoveryResumeCopy    RecoveryAction = "resume_copy"
	RecoveryRemoveTemp    RecoveryAction = "remove_temporary"
	RecoveryResumeCommit  RecoveryAction = "resume_commit"
	RecoveryRestoreBackup RecoveryAction = "restore_backup"
	RecoveryRemoveBackup  RecoveryAction = "remove_backup"
	RecoveryMarkCommitted RecoveryAction = "mark_committed"
	RecoveryManualReview  RecoveryAction = "manual_review"
)

type EntryObservation struct {
	TemporaryExists bool
	TargetExists    bool
	TargetMatches   bool
	BackupExists    bool
}

type RecoveryStep struct {
	Entry  int            `json:"entry"`
	Action RecoveryAction `json:"action"`
}

// PlanRecovery never proposes overwriting an unknown target. Any ambiguous
// filesystem state stops at manual review rather than guessing which file wins.
func PlanRecovery(transaction Transaction, observed []EntryObservation) []RecoveryStep {
	steps := make([]RecoveryStep, 0)
	for index := range transaction.Entries {
		var state EntryObservation
		if index < len(observed) {
			state = observed[index]
		}
		add := func(action RecoveryAction) { steps = append(steps, RecoveryStep{Entry: index, Action: action}) }
		switch transaction.State {
		case TransactionPlanned:
			add(RecoveryResumeCopy)
		case TransactionCopying:
			if state.TemporaryExists {
				add(RecoveryRemoveTemp)
			}
			add(RecoveryResumeCopy)
		case TransactionVerified:
			if state.TargetExists {
				if state.TargetMatches {
					add(RecoveryMarkCommitted)
				} else {
					add(RecoveryManualReview)
				}
			} else if state.TemporaryExists {
				add(RecoveryResumeCommit)
			} else if state.BackupExists {
				add(RecoveryRestoreBackup)
				add(RecoveryManualReview)
			} else {
				add(RecoveryResumeCopy)
			}
		case TransactionCommitting:
			if state.TargetExists && state.TargetMatches {
				if state.BackupExists {
					add(RecoveryRemoveBackup)
				}
				add(RecoveryMarkCommitted)
			} else if state.BackupExists && !state.TargetExists {
				add(RecoveryRestoreBackup)
				add(RecoveryManualReview)
			} else {
				add(RecoveryManualReview)
			}
		case TransactionCommitted:
			if state.TemporaryExists {
				add(RecoveryRemoveTemp)
			}
			if state.BackupExists {
				add(RecoveryRemoveBackup)
			}
		case TransactionRollingBack, TransactionFailed:
			if state.BackupExists && !state.TargetExists {
				add(RecoveryRestoreBackup)
			} else if state.BackupExists || state.TargetExists && !state.TargetMatches {
				add(RecoveryManualReview)
			}
		}
	}
	return steps
}
