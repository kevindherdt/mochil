# Vault text cache revalidation

Status: deployed with the 0.16.75 server image on September 7 at 18:47 UTC. Chronological not-deployed notes below describe the preparation stage.

The handler set five minutes of private freshness, but testing showed the shared JSON writer already overrides successful responses to `no-store`. The actual defect is conditional handling: it used only the catalogue revision as its ETag. Files edited outside Mochil can change between catalogue scans. The vault validator now includes the actual returned content hash and current writable status alongside the catalogue revision. Both 200 and 304 responses carry `no-store` and ETag, preserving the existing successful-response policy.

The regression fixture requests a page, verifies unchanged content returns 304, then replaces the file with different same-length text without scanning and verifies the old ETag receives 200 with the new content. The fixture operates only in a temporary directory. An initial fixture setup omitted the required library display name; this was corrected before validation.

This does not add file watching or avoid reading content during revalidation; the endpoint already read text before conditional handling. No live vault files were changed. This is not a claim that browsers previously cached these responses for five minutes.

## Exact editable text

The subsequent read/write audit found a real revision mismatch: generic document extraction removed a UTF-8 BOM and normalized CRLF, while the vault writer compares revisions against raw file bytes. Editable vault Markdown now returns its exact UTF-8 content, including line endings and any BOM. Ordinary document-library normalization remains unchanged. Invalid UTF-8 and size bounds still reject the read.

HTTP round-trip regression tests cover CRLF/frontmatter, UTF-8 BOM, and mixed line endings with Unicode. Each checks the returned content and revision against the original file, edits through the actual vault API, and verifies exact written content. Document, vaultwriter, and web package tests passed. Only temporary test files were modified; this change is not deployed.
