# Native wiki short filename links

Status: published in 0.16.82/code243. See `ANDROID_RELEASE_0.16.82_PREPARATION.md`; source-only notes below are chronological.

Android page lookup already checked local/root paths, but its short-name fallback compared only the display label. A note at `Projects/Roadmap.md` titled `Future plans` could not be opened by `[[Roadmap]]` from another folder. A different note titled `Roadmap` could incorrectly win.

The fallback now checks exact-case filename, then unique case-insensitive filename, before the existing display-label fallback. Local/root path precedence and explicit-path safety remain unchanged. Ambiguous filename matches return no result rather than falling through to labels or depending on scan order. Filename comparison uses region offsets instead of allocating a stripped basename per candidate.

Tests cover differing titles, conflicting display labels, ambiguous duplicate basenames, local context and exact-case preference. Both Android flavor targeted wiki suites passed (`/tmp/bobobooks-wiki-filename-links.log`). This is link-resolution coverage, not full Obsidian compatibility or physical-device acceptance. No layout changes were made.

## Web exact-case parity

The web resolver already used filenames rather than display titles, but its basename index was case-folded only. It now retains an exact-name map before the folded fallback, matching native behavior for distinct `Index.md` and `index.md` in different folders. Duplicate exact filenames still fail safely; local path priority is unchanged. The extra map is built with the existing vault index, not by scanning the vault on every click. It adds O(n) references; no memory or latency percentage improvement is claimed.

Web regression covers opposite scan orders, custom display titles, exact-case matches, ambiguous uppercase lookup and local context. Full frontend/module verification: `/tmp/bobobooks-wiki-link-parity-tests.log`. Both changes remain source-only after 0.16.81.

## Parse-once navigation

The native view model parsed a destination and then passed its stripped page name to a resolver that parsed again. `Report.md.md` could thus resolve to `Report.md`. The resolver now has a typed parsed-destination overload, used directly by the view model; the raw-string entry point parses once and delegates. Tests cover both paths with conflicting filenames and a heading/alias.

PDF dispatch also now checks the original filename extension, so a Markdown note named `Manual.pdf.md` is not intercepted as a PDF attachment. The PDF opening regression verifies that it falls through to note navigation without loading an attachment. The web reference parser does not strip the filename extension before its resolver, so that double-strip change is not needed there.

All native wiki tests in both flavors run through `/tmp/bobobooks-wiki-destination-tests.log`; these remain unpublished changes, not a new APK.

## Missing-note creation round trip

Follow-up inspection found that the missing-note proposal stripped one `.md` suffix and then used an idempotent extension helper. A link to `Report.md.md` therefore proposed `Report.md`. It now restores exactly the extension removed by parsing. Regression tests cover local and explicit-folder links and verify that the proposed newly created entry resolves from the original link.

Full native unit suites were rerun without cached tasks (`/tmp/bobobooks-wiki-full-regression.log`, successful), then both full flavor suites passed again after the creation fix (`/tmp/bobobooks-wiki-creation-regression.log`). No UI changes, media writes, APK build, or deployment were made for this follow-up. Physical-device acceptance remains outstanding.
