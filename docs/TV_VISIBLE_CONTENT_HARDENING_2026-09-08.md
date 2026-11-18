# Android TV visible-content hardening

Status: implemented in source; pending the next batched Android release.
The installed 0.16.83 APK does not contain this change.

Following the Impeccable native audit, TV empty-state detection now follows the
audio grid's actual content branch. Cached discovery tracks and root groups no
longer suppress the empty message inside an empty album or filtered track view.
Artist results follow the artist's albums. Discovery checks only its rendered
track feed, without allocating the bounded merged feed just to test emptiness.
Video detail emptiness no longer counts cached root shows.

Focus-membership memoization now includes selected artist and artist albums, so
an asynchronously arriving album list invalidates the membership result.
This preserves the existing single-key lookup and avoids building an identity
set for every loaded card. No mobile controls, layout, or styling changed.

## Verification

- Both complete Direct and Store debug unit suites passed in 14 seconds.
- Four new tests cover empty albums with cached root data, arriving artist
  albums, filtered favorites and the TV discovery feed.
- Module-size/responsibility gate passed.
- Test log: `/tmp/bobobooks-tv-empty-tests.log`.
- No Android devices connected. Remote focus restoration and rendered empty
  states still require device acceptance; unit tests do not prove visual polish.

The broader native audit's font-scale, overscan and viewing-distance checks
remain open. No APK was built or deployed for this small batch.
