# Vault expansion projection

Status: post-0.16.75 worktree, not packaged.

WikiTreeIndex already retained the sorted tree. Visibility previously split every path and rebuilt every ancestor prefix on every branch toggle. Ancestor paths are now computed once with the immutable index. Siblings share one ancestor list to limit retained duplicate strings. Expansion performs set membership checks without repeated path splitting, prefix joining, or sorting.

This trades a retained ancestor lookup for fewer transient allocations; no measured device speed or memory percentage is claimed. Tests cover repeated nested collapse/expand, requiring all ancestor folders to be expanded, natural ordering, hidden attachments, and searching collapsed pages. Both full Android unit-test suites and module checks passed. Sorting and visible UI remain unchanged.

## Attachment identity correction

Android's image index previously fell back to a matching basename even for a missing explicit path such as `Missing/cover.png`. It could display `Other/cover.png` instead. Explicit paths now resolve only through their relative/vault path candidates; failure does not substitute another folder's attachment. Bare `cover.png` shorthand retains the existing vault-order fallback. Existing relative/root candidate precedence is unchanged.

Tests cover missing folders, explicit current/parent-relative paths, backslash normalization, embed size suffixes, valid relative/root references, and shorthand. Both Android unit-test suites and module checks passed. This is an Android resolver correction, not a claim of complete cross-client Markdown compatibility. No APK was rebuilt.
