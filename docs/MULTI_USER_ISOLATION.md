# Multi-user isolation evidence

This matrix is the release gate for Mochil local accounts. Stable account
IDs, rather than mutable usernames, key every personal store. Library indexes,
metadata, podcast feeds, transcode caches, and server configuration are shared
administrator-owned resources by policy.

| Boundary | Account-owned state | Evidence |
| --- | --- | --- |
| Browser and Basic/OPDS authentication | Request identity and role | `TestSessionAndBasicAuthAttachStableRequestIdentity`, `TestOPDSRetainsBasicAuthForReaderClients` |
| Session revocation | Existing browser sessions | `TestDisabledIdentityRevokesExistingBrowserSession` |
| Android authentication | Random bounded device tokens | `TestAndroidLoginExchangesBasicCredentialsForDeviceToken`, `TestDisabledAccountDeletionPurgesEveryPersonalStoreAndCredential` |
| KOReader/KoSync | Pairing tokens and reading positions | `TestKoSyncProgressResolvesStableAccountWithoutCrossAccountReplay`, deletion corpus |
| Jellyfin compatibility | Random device tokens and video progress | `TestJellyfinTokensBindProgressToStableAccounts`, `TestJellyfinTokenStopsWhenAccountIsDisabled`, deletion corpus |
| Reading | Current position, history, dismiss/finish state, client locators | `TestRequestIdentityIsolatesReadingStateBetweenLocalAccounts`, account export and deletion corpora |
| Annotations | Highlights, notes, bookmarks, revisions and tombstones | annotation store cross-account tests, account export and deletion corpora |
| Audio | Progress, queues, favorites and playlists | account export and deletion corpora |
| Video | Progress and Continue Watching state | `TestProgressPersistenceAndAccountIsolation`, account export and deletion corpora |
| Collections | Definitions and members | account deletion corpus preserves the parallel owner account |
| Wiki | Pages, folders and history | wiki store cross-account tests, account export and deletion corpora |
| External library | Membership and per-chapter progress | `TestSharedProviderReferenceSurvivesOneAccountRemoval`, account export and deletion corpora |
| Administration | Global settings and mutation routes | `TestAdministratorOnlyPolicySeparatesGlobalAndPersonalMutations`, `TestOrdinaryAccountCannotReachGlobalControls` |
| Migration | Legacy administrator key and explicit recovery retirement | account store bootstrap, migration, and recovery tests |
| Export and deletion | Password-free export; disable-first, retryable purge | `TestAccountExportIsBoundedPasswordFreeAndIsolated`, `TestDisabledAccountDeletionPurgesEveryPersonalStoreAndCredential`, `TestInterruptedAccountDeletionRemainsDisabledAndCanBeRetried` |

The deletion corpus seeds two parallel accounts in every account-keyed store,
deletes one, verifies all of its content and Android/Jellyfin/KOReader tokens are
gone, and verifies the other account remains byte-logically available. A forced
durable-write failure separately proves that deletion rolls back the failed
store, retains the disabled credential, and succeeds when retried.

Run the gate with:

```sh
go test ./...
go test -race ./internal/accounts ./internal/reading ./internal/annotations ./internal/audio ./internal/video ./internal/externallibrary ./internal/wiki ./internal/web
```
