# Web wiki heading links

Pending after 0.16.78; no deployment for this change.

Wiki link rendering now retains the heading suffix in `[[Page#Heading]]` and `[[#Heading]]`. Same-page references jump without another page request; cross-page references pass their heading into the existing request-owned page load. The helper searches rendered article heading text using Unicode normalization and case-insensitive whitespace-normalized comparison, not dynamic selectors. It scrolls without animation and transfers keyboard focus without a second scroll.

Malformed encoded references remain inert. Duplicate headings select the first matching rendered heading. Block references, standard Markdown relative-page links, missing-heading feedback and more complete relative/ambiguous wiki-page resolution remain follow-ups; this is not full Obsidian link compatibility.

All 509 frontend tests and module checks passed. Chromium fixtures at 1440px and 390px verified scroll and focus with accented/CJK headings. An initial fixture omitted UTF-8 declaration and failed; declaring the fixture charset corrected that harness issue. Fixtures do not prove whole-app cross-page navigation or visual acceptance. Impeccable hardening guided malformed input and keyboard handling; no layout or theme was changed.

## Indexed page resolution follow-up

The web library now builds a dedicated page resolver when vault entries load. Same-folder resolution precedes root lookup, explicit `./` and `../` references stay relative, and unique basename fallback never chooses an ambiguous first match. Exact path casing wins; ambiguous case-insensitive matches stay unresolved. Only pages in the selected vault are indexed. Absolute URLs, control characters and paths escaping the vault are rejected; percent-encoded literal filenames are not decoded again.

This replaces repeated whole-page-list link matching with indexed lookups (index construction remains proportional to loaded entries). Tests cover order independence, relative/root priorities, ambiguity, case, vault/type isolation, Unicode and literal percent names. Missing/ambiguous-link feedback and full browser integration coverage remain follow-ups, not complete Obsidian compatibility. No deployment or layout change.

## Image resolution follow-up

The existing image resolver sorted relative/root candidates by catalogue order, so root artwork could override an image beside the note. It now gives relative candidates priority, keeps explicit dot-relative references from falling back to unrelated root files, rejects traversal above the vault, and refuses ambiguous basename/case-folded matches. The old test expecting the first of two `cover.png` images was updated to expect no arbitrary selection; new tests cover root/local order independence and explicit relative boundaries. Asset URLs remain constructed authenticated same-origin paths, never metadata-supplied URLs. No files were rewritten.

## Controller-level verification

The request-ownership fixture now supplies the real page resolver and heading helpers to the actual document-library controller. New tests exercise link clicks, cross-page requests with deferred heading jumps, same-page jumps without new requests, and switching vaults before an old heading request finishes. A stale response neither scrolls nor focuses the new library. Both cases passed in the full 517-test frontend suite; module checks also passed.

These are controller integration tests with lightweight DOM doubles, complementary to the earlier real-Chromium heading scroll/focus fixtures—not whole-app rendered acceptance. No production source or deployment changed in this verification pass.

## Standard Markdown navigation

Vault rendering now routes standard fragment links and relative `.md`/`.markdown` links through the same internal controller navigation, including percent-encoded heading text. The shared Markdown renderer exposes an optional link hook; renderers without it retain their existing safe-link policy. The hook receives an already escaped/rendered label. External URLs retain normal safe-link behavior, code remains literal, and malformed encoding, schemes, absolute destinations and control/backslash input are not converted into vault links.

All 519 frontend tests and module checks passed. The earlier standard-Markdown-link gap is addressed for these explicit note/fragment forms, not arbitrary attachments or every Markdown destination grammar. Not deployed.
