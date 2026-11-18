# Playback queue maintenance

Status: worktree changes after 0.16.75, not packaged.

Bulk queue insertion previously called a complete player lookup separately for each addition. Duplicate lookup now performs one descending-index pass against an addition-key set, removing non-playing duplicates without index shifts affecting later removals. The currently playing occurrence is preserved. Single additions reuse the same rule, fixing the case where finding the playing occurrence first left later queued duplicates untouched.

`publishQueue` now replaces its metadata lookup with entries still present in the player. Previously removals left old track/artwork metadata referenced until a full playback replacement or close. The new map remains proportional to the live queue, not all tracks visited during the ViewModel lifetime. Publishing incurs an additional linear map construction; this is a retention correction, not a claim of lower allocation per publish.

Tests verify reverse removal order, current-item preservation, empty-input behavior, and exactly 199 identity visits for a 200-item queue with one playing item regardless of a 200-key addition set. This bounds duplicate lookup, not Media3's internal removal cost or callback work. Both full Direct and Store unit-test suites passed. No measured device latency/RAM percentage is claimed.

Remaining issue: append operations can still grow the active queue beyond the 200-item initial-playback/snapshot window. A capacity policy must preserve user intent and provide feedback rather than silently dropping requested tracks. This audit deliberately does not introduce silent eviction.

## Contiguous removal batches

Adjacent duplicate indices are now grouped into half-open ranges and sent to `Player.removeMediaItems`, highest range first. This preserves the playing item and unrelated entries while reducing controller commands: the tested 200-item all-duplicate case with item 100 playing needs two removal calls instead of 199. Scattered duplicates can still require individual ranges. This is an operation-count improvement, not a measured playback-speed claim.

Regression tests cover the playing index at either edge, absent/current indices outside the queue, unrelated entries between duplicate runs, empty queues, and empty additions. Both complete Direct and Store unit-test suites passed after this change. These changes remain unbundled; no APK or live container was replaced.

## Preparation failure ordering

Single/bulk queue additions now construct all playback items before removing duplicates or modifying the metadata map. Full playback replacement likewise prepares items before replacing the map. A URL/media-item preparation exception therefore leaves the prior queue and lookup untouched. Empty bulk additions return without player commands or queue publication.

A source-order regression check covers these entry points, and both Android unit-test suites pass. This check does not simulate a connected MediaController or prove transactional behavior if a player command itself fails; device/integration coverage remains needed. No new UI or release artifact was introduced.
