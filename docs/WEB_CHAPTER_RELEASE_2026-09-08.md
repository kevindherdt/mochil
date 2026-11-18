# Web chapter recovery and navigation batch

Status: deployed September 8, 2026 at 14:21 UTC. Android artifacts/version unchanged at 0.16.82/code243. The result below supersedes preparation notes.

## Deployment result

Candidate image build completed successfully. The packaged assets passed chapter focus/reveal/close/scoped-Escape browser checks at 1440px and 390px (session 93388). The disposable container was stopped.

Live image: `sha256:e33c495a81eab6af8853db8287b25ff5c90e939230d854d0b59809977619433b`, started `2026-09-08T14:21:28.289360727Z`, healthy with zero restarts. All ten libraries loaded; the existing malformed EPUB warning is unchanged. Mounted Android APK hash matches the retained 0.16.82 archive. No media or unrelated containers changed. Physical-device/assistive-technology acceptance and the broad project goal remain open.

Includes scoped chapter keyboard focus/Escape/reveal, lazy-loader failure recovery, active audio-deck lookup and explicit seek ownership for chapter selection and Previous restart. See `WEB_CHAPTER_NAVIGATION_AUDIT_2026-09-08.md` and `WEB_CHAPTER_SEEK_OWNERSHIP_2026-09-08.md`.

All 560 frontend tests, local two-width chapter browser checks and module-size gate passed. Full Go tests/vet and distribution gate passed. Session 9361 is building image `opdshelf:web-chapters-20260908`; log `/tmp/bobobooks-web-chapters-image.log`. Revalidate the same handle before proceeding.

The chapter browser script now optionally reads embedded assets from an explicitly loopback disposable server, excluding live port 8099. Run it against the candidate before tagging/deployment. Stop that candidate afterward. Then verify live image, health, zero restarts, library readiness and unchanged APK hash.

Rollback image remains `opdshelf:android-0.16.82`, SHA-256 `bc649154b53742b4eb5ef44ea2a1a8466f244202e4cedff10c2770337c5a0507`. Direct APK remains SHA-256 `6c304b4746096b868bd277593071a398f36d7f0562e9aa6ab09fad9df71e36e7`. No Compose mount change or Android build is required.
