# Avoid full-history copies for unfiltered audio pages

Status: published at 18:31 UTC; see `AUDIO_COST_DEPLOYMENT_2026-09-08.md`.

`apiAudio` copied every progress record for the account and built a per-library
map for every request, even when no played/continue filter was selected. That
map was unused for filtering in ordinary views. Spoken-media response rows
already obtain their own progress; music rows intentionally omit resume state.

The full snapshot is now taken only when a played filter is present. Existing
filter cursor version checks remain intact. Plain podcast/audiobook rows retain
their progress and ordinary music browsing does not acquire new resume bars.
There is no change to stored history, sync ownership or the recently-played API.

## Controlled benchmark

One visible music file with 5,000 additional stored progress records, through
the real handler and JSON response path, two runs per version. This isolates
history overhead; it is not representative of a whole large-library scan/page.

| Path | Time/request | Allocated bytes/request | Allocations |
| --- | --- | --- | --- |
| Before | 1.38–1.80 ms | 2,468,087–2,468,307 | 107 |
| After | 9.44–10.75 µs | 7,844–7,845 | 54 |

The fixture removes about 2.46 MB of unnecessary allocation per request. The
gain in a real library depends on account history and other catalogue costs.
Logs: `/tmp/bobobooks-audio-history-{before,after}.log`.

Handler regressions verify ordinary/continue/unplayed/played views for music,
audiobooks and podcasts, including retained spoken progress and absent music
resume state. Full Go tests, vet and module gate passed:
`/tmp/bobobooks-audio-history-{go,vet}.log`.
