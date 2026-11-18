# Audio progress concurrent durability check

Added a file-backed regression for twelve simultaneous devices submitting
different positions against one revision. Exactly one update succeeds; the
others return the expected revision conflict. The resulting revision increments
once and reopening the progress store returns the complete winning record.

Twelve subsequent equivalent stale checkpoints from differently named passive
devices are acknowledged without changing serialized state or the store version.
Thus the winning position, attribution and timestamp survive both passive
acknowledgements and a store reopen.

Command: `go test -race ./internal/audio -run
'^TestConcurrentProgressHasOneDurableWinner$' -count=20`.
All twenty repetitions passed under the race detector.

This adds verification, not a runtime change. It proves this server-store
scenario, not physical-device offline delivery, playback handoff or all reading
sync paths. Existing numeric-validation tests remain separate coverage.
