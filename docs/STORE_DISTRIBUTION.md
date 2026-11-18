# Store distribution and extension policy

Mochil supports media interoperability through declarative connections. It
does not download, install, or execute third-party extension code in any build.

## Product position

Mochil does not provide media, endorse third-party repositories, or grant
permission to access copyrighted works. Users must only connect sources and use
content they are legally authorized to access. Repository authors remain
responsible for their code and service integrations. This notice is a clear
product rule, not a substitute for jurisdiction-specific legal advice.

## Distribution variants

- **Google Play and direct Android:** local media, Mochil servers, OPDS,
  podcast RSS, WebDAV, and declarative authenticated connectors. Neither build
  downloads or executes DEX, JAR, native libraries, or extension APK code.
- **Apple platforms:** use native readers/players and declarative or server-side
  connectors. Downloaded executable Mihon/Tachiyomi extensions are not part of
  the iOS, iPadOS, or tvOS architecture.

Android build variants retain distribution-specific update behavior, but both
omit third-party extension installers and runtime loading. Hiding a control is
not treated as an adequate boundary; the capability is absent from source,
dependencies, manifests, and packaged code.
Saved-publication synchronization is likewise narrowed to device-read OPDS
entries: the server rejects other providers or server execution, while Android
discards unsupported rows instead of retaining dormant executable-provider
state in its encrypted cache.

Generated Android build output and Kotlin workspace caches are excluded from
both version control and the server-image build context. The store-readiness
gate enforces these exclusions so stale dependency reports from local
experiments cannot re-enter a source snapshot or Docker build context.
The same gate scans both optimized APKs and the actual Store App Bundle. It
rejects retired extension identities, personal deployment hostnames, and the
direct-install updater in every packaged Store artifact; Store APK manifests
also receive `aapt2` permission inspection.
The migration-only `external_sources` field is constrained to exactly two
runtime owners: the strict config decoder that clears it before validation and
the Settings service that redacts it on read and removes it on save. The gate
fails if any third production file starts consuming that field, so backward
compatible startup cannot quietly become a dormant integration surface.

## Android signing boundary

CI builds the direct and store `optimized` variants with the Android debug key
only for tests, shrinking, lint, permission inspection, and prohibited-code
scanning. Those APKs are never public release artifacts.

Publishable `Release` APK or App Bundle tasks intentionally fail unless all four
signing variables are supplied from a protected environment:

- `BOBOBOOKS_KEYSTORE_PATH`
- `BOBOBOOKS_KEYSTORE_PASSWORD`
- `BOBOBOOKS_KEY_ALIAS`
- `BOBOBOOKS_KEY_PASSWORD`

The keystore and passwords must remain outside the repository and container
image. Before the first public Android release, configure protected CI secrets,
Play App Signing, a privacy-policy URL, data-safety answers, reviewer access,
and the permanent application identity. Never upload an `optimized` or other
debug-signed APK to a public download page or application store.

## Monetization boundary

Core access to personal local media and self-hosted servers should remain useful
without payment. Sustainable options include a paid client, optional store-billed
premium client features, or subscriptions for developer-operated services.
Digital features sold in store-distributed clients must use the applicable store
billing rules. Purchases must never imply that Mochil sells or licenses media
provided by user-selected sources.

Before public submission, publish a privacy policy, terms/acceptable-use policy,
support contact, data-safety disclosures, reviewer demo mode, and clear review
notes describing self-hosting and every source capability.
