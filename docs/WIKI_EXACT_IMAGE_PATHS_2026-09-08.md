# Wiki image identity: exact paths before folded lookup

Status: source change after deployed 0.16.79; no new APK or server deployment.

The attachment indexes previously stored only case-folded paths. On a case-sensitive vault, `Notes/cover.png` and `Notes/COVER.png` therefore made even explicit exact references unresolved. Page links already distinguished exact paths.

Both web and Android now keep an additional immutable exact-path map built once per vault index. Each contextual candidate uses its exact match first, then the existing unique case-insensitive fallback. An ambiguous local candidate still cannot silently select a root image. Duplicate entries with the same identity remain safe; conflicting identities for one exact path remain unresolved. Bare-name ambiguity, vault isolation and authenticated asset URL construction are unchanged. The web resolver additionally rejects DEL control characters, matching Android.

Cost: one extra map entry per indexed image; no media reads or per-embed tree scans. No measured speed or memory improvement is claimed—this is a correctness fix with bounded index overhead.

Verification: seven focused web resolver tests passed; both Android flavors' `WikiAssetIndexTest` suites executed successfully (Gradle 13s). Full frontend suite result recorded in `/tmp/bobobooks-exact-assets-frontend.log`. Existing mobile UI and personal media are unchanged.
