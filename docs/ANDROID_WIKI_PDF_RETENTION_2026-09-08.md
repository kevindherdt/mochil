# Wiki PDF cache retention

Status: published in Android 0.16.83 on September 8, 2026. No media-library changes.

## Verified behavior

PDF cache identities include server URL, device/account identity, library, entry
and revision. A valid cached PDF with a nonempty matching revision is returned
without a network request. This is a disposable bounded cache, not a guarantee
that all vault attachments are downloaded or permanently retained offline.

## Fixed failure path

Previously a new attachment pruned older PDFs before attempting its download.
A failed request could therefore remove a usable offline copy without replacing
it. Pruning now follows validation and atomic publication of the new PDF. Cache
hits also enforce the count limit, recovering a cleanup interrupted after commit.
Only this component's opaque generated filenames are eligible for pruning.

The existing free-space check runs before downloading. Temporarily keeping the
old quota plus one incoming file trades additional transient storage for preserving
usable data. A lack of staging space fails without evicting prior PDFs.

## Regression coverage

At a one-file quota: cache a PDF, fail another download with an HTML response,
verify the first remains byte-identical and reopens without networking, then
successfully download the replacement and verify quota cleanup. Existing tests
cover malformed/oversize/truncated responses, cancellation and unrelated files.

Tests use real temporary files and controlled HTTP responses, not an Android
device or a live account. Both full Direct and Store unit suites passed (13 seconds,
`/tmp/bobobooks-wiki-pdf-retention-tests.log`), as did the module-size gate.
