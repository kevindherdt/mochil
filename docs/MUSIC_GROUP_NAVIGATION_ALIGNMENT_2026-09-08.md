# Music grouped A–Z navigation

Status: published with 0.16.85 on 2026-09-08 at 17:03 UTC. All grouped-navigation,
edition-ordering, terminal-pagination and compilation-search fixes described
below are included. Earlier pending statements are development chronology.
See `ANDROID_PENDING_BATCH_VERIFICATION_2026-09-08.md` for release evidence.

The navigation endpoint previously counted album titles, whereas grouped browsing
uses edition/folder identities. Same-named albums therefore shifted later letter
offsets. Artist navigation sorted tracks by performer while labeling groups by
album artist, producing different positions from the displayed artist list.
Its search also matched track metadata rather than the group label/subtitle.

Grouped browsing and A–Z navigation now share one projection for identity,
filtering and natural ordering. Equal group names tie-break by stable group ID,
so separate requests/pages cannot randomly exchange same-titled editions.
Track navigation behavior is otherwise unchanged.

HTTP-handler regressions compare group pages and letter offsets/counts across
albums/artists, duplicate album names in separate folders, performer versus album
artist, label queries, hidden-track-only queries and empty results. Single-item
pages confirm stable edition order. Full web Go tests and module gate passed.
Log: `/tmp/bobobooks-audio-navigation-tests.log`.

No client layout changes or claims about scroll-highlight rendering. This fixes
the server contract consumed by Android grouped jumps; device acceptance remains.

## Artist detail follow-up

Artist album lists now use the same natural-name/stable-ID ordering helper as
root group lists. Previously their equal-named editions still depended on map
iteration order. Fifty repeated/reordered-input checks confirm artist detail IDs
match root edition order without merging same-titled folders. Full Go tests,
Go vet and module gate passed. Logs: `/tmp/bobobooks-group-order-{go,vet}.log`.
The group/navigation batch remains source-only pending deployment.

## Library shrink pagination regression

Grouped pagination previously reset an offset beyond the current group count to
zero. A rescan removing groups between requests could therefore replay page one
into an existing scrolling list. Out-of-range positive offsets now clamp to the
end, returning an empty JSON array with no next-offset header. Negative offsets
still start at zero and page-size limits remain unchanged.

Handler tests cover exact-end and beyond-end requests for albums and artists,
including filtered and empty results, alongside stable single-item pagination.
Full `go test ./...`, `go vet ./...` and the module boundary/size gate passed.
Logs: `/tmp/bobobooks-group-pagination-{go,vet}.log`.
This follow-up is source-only; no APK or running container was changed.

## Complete compilation search results

Group search previously filtered tracks before building albums. Searching a
compilation contributor could therefore reduce its count, replace its displayed
artist and select different artwork even though opening it loaded the full album.
Groups are now assembled completely before selection. Contributor-name matches
remain supported, as does searching the final displayed `Various artists` label.
Track titles still do not match group search. Both grouped pages and navigation
use this projection, so their offsets continue to agree.

A compilation regression checks count, ID, subtitle and representative artwork
across contributor and displayed-label searches. Full Go tests, vet and the
module gate passed; logs are `/tmp/bobobooks-group-search-{go,vet}.log`.
This is also pending deployment and does not change client layout.
