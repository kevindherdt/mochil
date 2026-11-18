# Wiki image cache instance safety

Status: implemented and tested in source; pending next batched Android APK.

The offline image store previously synchronized on each instance. Multiple
instances use the same app-private directories, so one could prune another's
in-flight temporary file during overlapping activity/view-model lifetimes.

Merge/prune and URL lookup now share a process-wide storage lock. Network loading
stays outside this lock. Existing per-image and total byte budgets, owner/vault
partitioning and atomic replacement remain unchanged. A File-based constructor
allows actual filesystem tests without mocking Android Context.

Three new tests exercise retained revisions after unavailable optional downloads,
obsolete-file pruning, overlapping instances writing 60 batches with both desired
assets retained and no temporary leftovers, and owner/vault isolation.
The concurrency test is a stress regression, not an exhaustive scheduler proof;
serialization is enforced by the shared monitor in production code.

Both full Direct/Store debug unit suites passed in 15 seconds. Module gate passed.
Log: `/tmp/bobobooks-wiki-assets-storage-tests.log`.
No user files or deployed cache were modified. No layout changes or APK build.
