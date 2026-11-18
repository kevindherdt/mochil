# Audio journal instance overlap

Status: source-only after 0.16.80.

Audio journal mutations were synchronized on individual instances, although all instances share the same encrypted outbox. That does not protect read/modify/write transactions across overlapping instances. Video already uses a process-wide journal lock.

Audio now shares a transaction lock across instances. Loading, merging/pruning and persistence remain inside it. Storage callbacks allow the actual journal implementation to be tested with shared in-memory bytes; the production constructor retains the same encrypted store name, key alias, schema and entry limits. No user data migration is needed. This protects one Android process, not hypothetical independent processes sharing storage.

Tests pause one instance during its read, verify another cannot read concurrently, then confirm both checkpoints survive. A second test verifies an old acknowledgement cannot remove a newer checkpoint saved through another instance and that unchanged reads do not persist again.

Both flavor targeted journal suites passed in 12 seconds (`/tmp/bobobooks-audio-journal-lock-tests.log`). Full Direct/Store unit verification was launched in session 45767, log `/tmp/bobobooks-post80-android-tests.log`. No APK build, deployment or physical-device validation is included.
