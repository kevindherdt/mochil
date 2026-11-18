# Audio lifecycle checkpoint durability

Status: published in 0.16.86/code 247. See `ANDROID_0_16_86_RELEASE_VERIFICATION.md`.

## Verified gap

`PlaybackService.onPositionDiscontinuity` sent the departing item's position
when the playlist index changed, but used the default non-durable submission.
A failed network send could therefore lose that final checkpoint: later pause
or shutdown writes describe the new current item, not the previous episode.

The departing-item submission now commits to the encrypted progress outbox
before delivery. `onDestroy` also checkpoints current progress before cancelling
the sender instead of saving only the player snapshot. Existing task-removal,
pause and explicit-seek checkpoints remain intact. An empty player still clears
the playback snapshot; this does not reopen a player the user dismissed.

## Verification and limits

A new regression test uses the actual audio HTTP client and serialized journal:
two episode checkpoints survive a 503 response, a recreated journal replays
both after recovery, and only acknowledged records are removed. Both complete
Android debug unit suites and the module-size gate passed.
Log: `/tmp/bobobooks-audio-durable-recovery-tests.log`.

The test proves outbox/HTTP recovery, not Android OS callback delivery. Lifecycle
wiring was inspected in source; physical interruption testing remains pending.
Force-killing a process need not invoke `onDestroy`, and this patch does not
make periodic positions durable or remove the journal's existing 64-item cap.
Storage commit failures continue to be recorded in diagnostics. Existing
revision conflict, passive-progress and explicit-seek expiry policies are not
changed by this patch.
