# Release gate integrity follow-up

The current Direct and Store optimized APKs pass the retired-extension,
personal-hostname, and Store permission checks. These checks are scoped
technical evidence, not a promise of Play Store acceptance.

Review found that archive-string scanning could hide an `unzip` failure behind
a no-match result from `grep`. A damaged Direct APK could consequently pass
without actually being inspected. The gate now validates archive integrity
with `unzip -tqq` before inspecting packaged content and rejects unreadable or
corrupt archives explicitly.

Regression coverage invokes the real shell gate against empty and truncated
Direct APK fixtures, checks failure and the specific integrity diagnostic, and
removes only its generated temporary fixture directory. Both existing optimized
APKs still pass the strengthened gate. Shell syntax validation passes. No APK,
server configuration, media file, or signing material was changed.

This does not replace signature verification, complete manifest validation,
device installation tests, or the remaining physical Android TV checks.

Follow-up finding: the Android CI job currently assembles optimized packages,
but a clean checkout has no private persistent signing key and the workflow
does not provision one. The signing gate correctly rejects that condition.
CI needs a clearly test-only temporary signing identity passed explicitly to
the optimized build, never a fallback change in production Gradle configuration
and never the personal update key. Such CI packages must not be published as
user updates.

The CI workflow now generates a two-day test-only key under the runner's
temporary directory, refuses an existing target, restricts creation permissions,
and explicitly supplies its path only to the verification build step. No
production Gradle fallback was added, and the job has no APK upload/publish step.
A locally generated temporary identity passed `verifyOptimizedSigning` with
configuration caching disabled; that test key was then removed. No optimized
APK was rebuilt during this test, avoiding replacement of the live download.
Its SHA-256 remains `559d17d7ef60f241471d7547ab6776335beeb9ece55e3d5b655036f9703525a6`.
The full workflow still needs execution on an actual clean GitHub runner;
local signing-gate verification does not prove CI packaging end to end.
