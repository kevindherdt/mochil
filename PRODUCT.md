# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

The primary user is a self-hoster managing and reading a personal ebook and
comic collection. The product should also remain approachable for other
self-hosters whose libraries live in ordinary filesystem folders or Calibre.

## Product Purpose

Mochil turns existing storage into a browsable OPDS and web reading library
without making its own database or folder structure the canonical source. A
successful setup lets someone connect storage, see a polished cover library,
read edge-to-edge in the browser, and add or maintain libraries without editing
JSON.

## Positioning

Mochil is a small, storage-native alternative to heavyweight library servers:
it indexes what users already organize in folders or Calibre and
keeps the source system authoritative.

## Operating Context

Mochil runs as a small Go service, commonly in Docker or an Incus container,
behind a reverse proxy. Readers use its web library or connect an OPDS client
such as KOReader. Administrators connect storage and define which folders are
Books, Light Novels, Manga, Comics, or mixed collections.

## Capabilities and Constraints

- Existing OPDS 1.2 feeds and acquisition URLs must keep working.
- Server-side credentials (provider API keys, account passwords) remain
  server-managed secrets and are never revealed by the settings UI.
- The administration UI manages library definitions and scan intervals without
  requiring hand-edited JSON. Repository/folder discovery, health, and manual
  rescans are planned follow-up work.
- The web library supports cover-led browsing and downloads.
- The reader uses Foliate's web reading components for EPUB and supports an
  edge-to-edge reading surface. Comic reading remains compatible with CBZ page
  content and can expand independently.
- Storage remains canonical; caches and indexes are disposable.
- Storage integration keeps an adapter boundary between filesystem and Calibre
  sources.
- The existing dependency-light Go server remains the application host.

## Brand Commitments

The product name is Mochil, named for the owner's cat Mochi. (Its predecessor,
BoboBooks, was named for the dog, Bobo.) The interface should be minimalistic, calm, and
well-crafted, with the practical clarity of a personal library rather than the
density of a media-server control panel. Calibre is a functional reference for
cover browsing, not a visual template.

## Evidence on Hand

- A working library with more than two thousand publications.
- Existing OPDS feeds, search, downloads, authentication, and multi-library routing.
- No existing logo, cover-art direction, testimonials, or commercial claims.

## Product Principles

- Existing storage is the source of truth.
- Setup should explain itself and avoid configuration-file work.
- Reading gets the whole screen; administration stays out of the way.
- Small personal deployments remain simple while adapters and library boundaries
  allow growth.
- Security-sensitive credentials stay outside browser-managed configuration.

## Accessibility & Inclusion

Keyboard navigation, visible focus, readable contrast, reduced-motion support,
responsive layouts, and adjustable reader typography are required. The web
library must remain useful when cover images are unavailable.
