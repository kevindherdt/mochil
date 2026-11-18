# Preserve unreadable video progress

Status: source-only after 0.16.80; no APK or deployment.

Video's journal previously used best-effort encrypted loading and a decoder that returned an empty list on invalid JSON, oversized payloads or invalid rows. The next mutation could replace the unreadable checkpoint set.

Journal transactions now opt into strict encrypted loading and strict bounded decoding. Read errors propagate to the reporter's existing guarded restore/replay paths; writes and acknowledgements return failure without persisting. The best-effort decoder remains available for callers whose contract requires it. Size, row-count, owner and field validation are unchanged.

Tests supply a throwing loader, truncated JSON and an oversized payload, verifying reads fail and neither put nor acknowledgement writes anything. Existing valid journal, merge, reporter and recovery tests remain applicable. Both flavor targeted verification completed successfully (`/tmp/bobobooks-video-journal-strict-tests.log`). Full post-change tests are recorded separately; no hardware Keystore testing is claimed. Corrupt data is preserved, not repaired.
