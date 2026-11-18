# Podcast duration parsing boundary

Status: published at 18:22 UTC; see `AUDIO_VALIDATION_DEPLOYMENT_2026-09-08.md`.

The architecture review found feed duration parsing embedded in the audio HTTP
projection. Its ParseFloat accepted NaN and infinity, and colon accumulation
could overflow. These values are not JSON-encodable, so malformed external feed
metadata could break a catalogue response rather than merely omit duration.

Parsing now belongs to `internal/podcast/duration.go`. Episode projections and
audio-progress duration lookup share this domain function. It retains ordinary
seconds, fractional seconds and colon-separated input, but treats non-finite,
negative, malformed and overflowing values as unknown (zero). Feed records and
media files are not rewritten. No speculative duration cap or UI change is added.

Table tests cover valid and invalid inputs; a projection regression confirms
bad durations no longer prevent episode-list JSON encoding. Full Go tests,
vet and the module-size gate passed. Logs:
`/tmp/bobobooks-podcast-duration-tests.log` and
`/tmp/bobobooks-podcast-duration-vet.log`.

This separation follows responsibility, not an arbitrary file-length target.
No general parser framework or new dependency was introduced.
