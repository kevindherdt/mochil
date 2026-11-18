# Audio progress conflict exchange verification

Verified the real Android AudioProgressClient against a synthetic loopback HTTP
server. No personal service or progress records were used or changed.

Three new regression tests prove the request sequence, not only a pure policy:

- GET revision 1, PUT conflicts, GET newer remote position: acknowledge the
  obsolete local position without issuing another PUT.
- Same exchange with remote still behind: retry once using refreshed revision 2
  and the local position, without claiming an explicit seek.
- Second PUT also conflicts: return failure for durable recovery; a subsequent
  delivery starts with GET rather than reusing the contested cached revision.

Both complete Direct and Store debug unit suites passed in 11 seconds.
Log: `/tmp/bobobooks-audio-conflict-exchange-tests.log`.

No production behavior changed: these tests strengthen existing protection.
This does not prove all multi-device synchronization, offline journal delivery,
or end-to-end server ownership semantics. Physical device testing remains open.
