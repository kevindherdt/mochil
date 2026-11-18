# Navigation and scan deployment

## Verified live state

The `opds-opdshelf-1` container started at 2026-09-08T17:43:02Z and runs
`sha256:39d61959ee3d91444a0bb1e245af89b4f0125296f18f7e11071aa43138f5cbb2`.
This matches `opdshelf:navigation-scan-20260908` and the tested candidate.
Post-deployment inspection reports healthy, zero restarts and no OOM kill.
All ten libraries loaded. The existing malformed EPUB warning remains; no
library files were changed to suppress it.

Android remains 0.16.85/code 246. The archived direct APK SHA-256 is unchanged:
`6c7e240bf3423c1bc9f12bc96377b2bf35ff8c131ae710aaac20399d886d2e7b`.
TV navigation changes still require a subsequent native release.

## Verification scope

- Full Go tests and vet completed successfully.
- Frontend tests: 564 passed, zero failed.
- Actual navigation modules passed synthetic browser interaction checks at
  1440px and 390px: jumps, scroll selection, one active marker and hiding.
- Candidate update/download and playlist revision smoke checks passed using
  synthetic credentials and data; real-server wiki smoke passed at both widths.
- The browser interaction fixture is not a full visual or hardware acceptance
  review. No whole-library scan speedup percentage is claimed.

Logs: `/tmp/bobobooks-navigation-scan-{go,vet,web,browser}.log` and
`/tmp/bobobooks-navigation-scan-image.log`. Temporary logs may not survive
environment replacement; the deployed image identity above is the durable
release reference. Prior image tag `opdshelf:grouping-performance-20260908`
remains available as a rollback reference.

## Included changes

- Shared web bucket classification matches server rules, including articles
  and Unicode edge cases; it does not change the approved layout.
- Audio scans skip explicit metadata lookups for unsupported sidecars while
  preserving supported-media checks and artwork lookup.
- Fresh audio filename fallback titles strip original-case extensions.
  Unchanged cached metadata is not forcibly rewritten or reprobed.

During restart investigation, Pulse—not Mochil—was observed crash-looping.
Pulse was not modified; its issue is outside this deployment.
