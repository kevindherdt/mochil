# Repeat offline wiki image saves

Status: source change after 0.16.84; pending a later batched release.

Offline mirroring previously downloaded all planned images again. It now checks
the owner/vault cache and skips only nonempty, in-budget files whose ID, path and
nonblank revision match. Unknown revisions, changed paths/revisions, missing files
and invalid sizes still require downloading. Existing final count/byte pruning is
unchanged and download-failure reporting now counts only requested downloads.

The file-backed regression fixture has 32 unchanged images. Repeat planning feeds
zero entries to the batch loader (previously 32) and retains all 32 cached files.
This eliminates those image requests/transfers and replacement writes; it is not
a measured whole-vault latency or memory claim. Markdown pages still refresh.
Other tests cover identity/owner/vault isolation and empty/oversized cache files.

Both Direct/Store debug unit suites and module gate were run; test log:
`/tmp/bobobooks-wiki-reuse-tests.log`. No APK was built for this individual change.
