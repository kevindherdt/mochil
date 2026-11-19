# Changelog

Written for someone upgrading, not for someone reading the diff. The detailed engineering history
lives in `internal/changelog/changelog.json`; this file is what a user needs to know.

## 0.1.0

First public release.

- One container image, three folders, one command to install. No database: a Go server serves the
  catalogue and the web app from a single binary.
- Books, comics, manga, light novels, audiobooks, podcasts, music, films and TV series in one
  library, from folders you already have.
- Web reader for EPUB and PDF, comic reader, video and audio players, on the same pages it browses.
- Android phone and TV app, and a KOReader plugin, both pairing with the server rather than storing
  your library credentials on the device.
- Optional pieces stay off until you ask for them: online metadata, narration, memory, the wiki.
- AGPL-3.0: free to run, change and share, with the changes staying open if you serve it to others.
