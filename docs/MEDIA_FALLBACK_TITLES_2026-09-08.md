# Media fallback filename titles

Status: published in the 17:43 UTC navigation-scan server batch.
See `NAVIGATION_SCAN_DEPLOYMENT_2026-09-08.md` for deployment evidence.

Audio identification accepted uppercase extensions but tried to remove the
lowercased extension from the original filename. `My_Song.Live.FLAC` therefore
retained `FLAC` in its fallback title. It now removes the actual filename suffix
before producing readable words. Extension matching and container labels remain
case-insensitive; original paths, identity hashes and revisions are unchanged.

Audio and video each retain their fixed title-separator replacer rather than
constructing it for every identification. These tiny helpers stay in their own
format domains; no general-purpose parser or new cross-media dependency is added.

Mixed-case FLAC/MP3/M4B tests verify title cleanup, path retention and repeatable
identity. Full Go tests, vet, focused parallel race check and module gate passed.
Logs: `/tmp/bobobooks-media-title-{go,vet}.log`.

This affects fresh identification. Cached, unchanged metadata is intentionally
not rewritten or globally re-probed; embedded titles still take precedence.
No media files were changed and no new APK was required.
