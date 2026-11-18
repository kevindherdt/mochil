# Sleep timer during buffering

Status: fixed in source, pending the next batched APK; not in installed 0.16.83.

PlaybackService previously consumed an expired timer before checking `isPlaying`.
When a stream was buffering or temporarily suppressed, that check was false and
the timer disappeared without clearing playback intent. Audio could subsequently
start after the configured sleep deadline.

Expiry now unconditionally calls pause through a small testable timer action.
Pause is safe when already paused and clears pending playback intent. The timer
is still consumed once, so later explicit playback is not repeatedly interrupted.
No streaming quality, automatic queue-transition behavior or UI was changed.

Verification: both full Direct and Store debug unit suites passed (13 seconds),
including two new tests for buffered playback intent, one-time expiry, later
explicit play and unexpired timers. Module gate passed. Log:
`/tmp/bobobooks-sleep-buffering-tests.log`.

Tests use injected timer/pause actions; real ExoPlayer buffering and audio-focus
behavior have not been exercised on hardware. No APK build or deployment yet.
