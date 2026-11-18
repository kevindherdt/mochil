# Audio scan sidecar metadata lookups

Status: published in the 17:43 UTC navigation-scan server batch.
See `NAVIGATION_SCAN_DEPLOYMENT_2026-09-08.md` for deployment evidence.

The filesystem audio scanner previously called DirEntry.Info for every file,
then rejected unsupported extensions and AppleDouble files in Identify. A
focused scan-entry helper now checks the existing supported-format map and
AppleDouble prefix before requesting file metadata.

This removes one explicit Info call per ignored sidecar. It does not eliminate
directory enumeration, guarantee a particular syscall count on every platform,
or imply a measured whole-scan speedup. Cover files remain available to artwork
lookup; none are deleted or modified.

The walk still handles symlinks/directories first. Accepted media still receive
file metadata checks, path/size validation, probing and revision reuse. Errors
from metadata lookup on supported audio continue to propagate.

Counting fake directory entries verify zero Info calls for covers, lyrics,
temporary files and AppleDouble entries, one call for every supported format,
uppercase acceptance and error propagation. Full Go tests, vet and module gate
passed: `/tmp/bobobooks-audio-scan-entry-{go,vet}.log`.
