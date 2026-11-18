# Release-readiness checkpoint

## Verified

`check-store-readiness.sh` passed against the current source and the archived
0.16.82 Direct and Store APKs. This verifies the checks implemented by that script,
not Play Store approval or the contents of a future APK.

The source gate includes retired executable-extension identities, dependencies,
discard-only legacy configuration, private build-context exclusions and flavor
separation of the APK installer. No removed integration needs to be restored.

## Pending native batch

- Video watched-state updates follow media identity rather than whichever show
  happens to be open when a response arrives.
- Search invalidates previous responses immediately and suppresses delayed work
  after navigation; pagination jobs are cancelled when leaving their context.
- Stale mutation failures do not interrupt a different catalogue request.
- Wiki offline cache lookup skips empty caches and uses compatible cheaper hex
  encoding; partial refresh image planning includes retained saved notes.

Both Android flavor unit suites and the module-size gate passed for these source
changes. They are not present in the archived 0.16.82 APKs. No APK was rebuilt at
this checkpoint.

## Next security review

The manifest exports the playback service with a binding permission. The service
returns its session from `onGetSession`, and creates MediaSession without an
application callback. The pinned Media3 dependency is 1.11.0.

Review actual dependency behavior and controller command admission before changing
this boundary. Android's [MediaSession callback reference](https://developer.android.com/reference/androidx/media3/session/MediaSession.Callback)
documents default controller admission and trusted-controller command behavior.
The source observations alone do not establish that another application can bind,
inject media, receive credentials, or control playback without authorization.
Preserve notification, Bluetooth, system and in-app controls when testing any
hardening. Do not infer a vulnerability solely from `exported=true`.

### Exact dependency inspection

Inspected Google's published `media3-session-1.11.0-sources.jar`, matching the
pinned dependency, rather than relying on older Media3 examples. The source's
`MediaSession.Callback.onConnectAsync` uses `AcceptedResultBuilder(session,
controller)`. That builder grants untrusted controllers read-only commands and
trusted controllers the normal session/player commands. The default media-item
resolver accepts supplied items only when they include a local configuration;
command admission still precedes that path.

The dependency's trust definition includes this app's UID, the system UID, media
control/status-bar permissions, and enabled notification listeners. Thus the lack
of an app callback is not evidence that arbitrary apps can issue playback commands.
No confirmed unauthorized playback or credential disclosure was found by this
source inspection. Real cross-UID binding and token access were not exercised.

Potential stricter policy: reject untrusted readers too, if privacy requirements
call for it. Do not silently impose that compatibility change based on outdated
claims that every controller receives every command. Notification-listener access
is user-granted and must not be confused with an unauthenticated remote connection.
No playback access behavior was changed at this checkpoint.

## Still outstanding

Physical Android TV focus/performance acceptance; broader wiki non-image offline
attachments and revision-history policy; remaining cross-platform feature and
design acceptance. The broad project goal is not complete.
