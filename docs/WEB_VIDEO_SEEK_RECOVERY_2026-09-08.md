# Video seek recovery follow-up

Status: source-only after 0.16.80; not deployed.

The video player cleared its explicit-seek flag at submission, before the network acknowledged it, and swallowed failures without resetting the timed-save threshold. A failed rewind could therefore lose its explicit intent on the next save. Delayed same-item conflict responses could also update ownership after a newer seek.

`video-progress.js` now owns serialization, passive ownership, seek generations, throttling and retries independently of the player controls. Explicit seeks survive transport failure; failures reset the timed-save threshold. Old same-item responses cannot consume or suppress a newer claim. Remote revisions are still adopted, and passive clients still yield to newer/finished remote progress. Server-known duration and keepalive writes are preserved. Local claim tokens never enter the HTTP request.

The player routes seeking/play/save events through this module. Close still samples its final checkpoint before relinquishing the active item and pausing; no UI, playback source or subtitle changes were made.

Focused tests cover offline retry, delayed success/conflict with a newer seek, remotely finished passive protection and explicit seek recovery, plus existing player contracts. Full frontend suite and module gate are recorded in `/tmp/bobobooks-video-seek-tests.log` and the verification command. No native or physical-device behavior is claimed by these web tests.
