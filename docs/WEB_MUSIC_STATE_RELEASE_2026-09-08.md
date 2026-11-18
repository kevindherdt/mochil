# Web/server music-state update

Status: deployed September 8, 2026 at 15:24 UTC; Android remains 0.16.83/code244.

## Deployment outcome

Image `sha256:b6cc6cab668c3d81766dd81f6d689a22348e6fb247b160a80c50335c8c98cc11`,
started `2026-09-08T15:24:39.608637292Z`, healthy with zero restarts and all ten
libraries loaded. Mounted Direct APK hash is unchanged. The sole scan warning is
the previously known malformed EPUB; no media was changed.

Full Go tests/vet, 563 frontend tests and module gate passed. Candidate checks
passed for authenticated playlist mutations and APK download/HEAD/range, real
wiki persistence/navigation at 1440px and 390px, playlist forms at both widths,
and delayed-save playlist dragging. Packaged music-state asset SHA-256 matched
the tested local source. Disposable containers were stopped after testing.

The preparation history below is retained for traceability. No native music
changes were included in the APK served by this web-only deployment.

Scope: external playlist refresh ordering and removal of redundant queue
validation. No visual changes, media-file changes, or APK rebuild.

Pipeline session 70941 runs full Go tests/vet, all frontend tests, the module gate,
then builds `opdshelf:music-state-20260908`. Inspect this session before continuing;
do not start a duplicate build.

Candidate checks must cover playlist behavior, wiki/browser regressions and the
unchanged 0.16.83 authenticated APK download. Deploy only after those pass.

Rollback image is `opdshelf:android-0.16.83`,
`sha256:487626ef620cb7b8e95df5a88ab61867bd7f933707dfaeeda756c1854e40a714`.
Direct APK SHA-256 remains
`a7e1d9af68055cfa0c78cfdb51db77ff95587114e5ad59b0a84181d8924d7bee`.

Native favorites request ownership, cached favorites across library switches,
music group Back loading state and root-pagination isolation are unshipped source
work for a later APK batch. They are not included in this web/server deployment.
