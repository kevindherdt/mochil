# Multi-user account decision

Status: approved. No public registration; the current environment account is
the initial administrator, and only administrators may create accounts.

## Current boundary

Mochil now authenticates persisted accounts with stable IDs and roles.
Browser, Basic/OPDS, Android, KoSync, and Jellyfin authentication resolve a
request-scoped identity rather than deriving state ownership from the configured
environment username. The environment credential remains a bounded recovery
path for the initial administrator.

The durable stores are already mostly ready for isolation. Reading positions
and history, audio and video progress, queues, favorites, playlists,
collections, wiki pages, and KOReader device tokens accept an account key.
Library definitions, scans, metadata review and overrides, transcode caches,
podcast subscriptions/downloads, health, jobs, and server settings are global.
Podcast played state is account-owned through audio progress, while feed and
download retention policy remains global.

KoSync protocol usernames and random, persisted Jellyfin device tokens now
resolve the same immutable account IDs. Disabled accounts immediately stop
authorizing existing Jellyfin tokens, and account-specific progress cannot be
replayed into another account.

## Recommended policy

- The existing environment account is migrated as the initial administrator.
- Administrators create, disable, and reset local accounts; there is no public
  self-registration.
- Sessions store an immutable account ID plus role. Display-name or username
  changes do not move or orphan account data.
- Browser and Basic/OPDS authentication resolve the same account identity.
- Library browsing is shared. Reading/playback state, queues, favorites,
  playlists, collections, wiki pages, and device credentials remain private to
  their account.
- Settings, library mutation, scans, metadata review/apply, global podcast
  subscriptions and retention, job controls, and account administration require
  the administrator role.
- Disabling an account revokes all of its browser and device sessions without
  deleting its data. Data export and deletion remain separate explicit actions.

## Migration sequence

1. [x] Add a bounded, versioned account store with stable IDs, normalized unique
   usernames, password hashes, roles, disabled state, and atomic persistence.
2. [x] Import the current environment credential as the first administrator without
   changing its existing account key. Keep environment authentication as a
   recovery/bootstrap path until the administrator has confirmed the persisted
   account.
3. [x] Put authenticated identity into request context and bind every browser
   session to an account ID. Replace the global `readingAccount()` fallback with
   that identity at one central boundary.
4. [x] Gate global mutation routes with an explicit administrator capability and
   add denial tests for every route family.
5. [x] Bind KoSync usernames/device tokens and Jellyfin tokens to the same stable
   account IDs, with cross-account replay tests.
6. [x] Provide create, disable/session revocation, password reset, recovery
   confirmation, bounded personal-data export, and permanent account deletion.
   Deletion requires a disabled account and exact username confirmation, purges
   every account-owned store and device credential before removing the login,
   and remains safely retryable after a storage failure. Passwords and device
   credentials are never returned by APIs or included in exports.
7. [x] Add cross-account integration tests covering every account-keyed store,
   OPDS/Basic authentication, session revocation, migration, and administrator
   authorization before deployment.

The maintained evidence matrix is in `MULTI_USER_ISOLATION.md`.

## Approved policy

The owner approved the recommended policy above: no self-registration, the
current account becomes administrator, and global controls remain
administrator-only. Wiki collaboration and sharing must follow this identity
foundation rather than inventing a separate user model.
