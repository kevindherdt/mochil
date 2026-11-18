# Wiki frontmatter/task boundary consistency

Pending after 0.16.78; no APK or deployment.

Property extraction searched for the first `\n---` prefix and abandoned frontmatter if that prefix was not a complete delimiter. Task mutation used a separate exact-delimiter expression that could continue to the actual closing delimiter. With malformed metadata containing a prefix line and a task marker, the renderer could expose a task that the mutation path skipped, shifting the user's selected task ordinal.

Extraction now uses the same exact frontmatter range as mutation. Regression coverage checks delimiter-prefix lines and task-like metadata with LF and CRLF, ensuring only the body task renders and only that marker changes. Both Android debug unit suites and module checks passed (12-second Gradle invocation).

This is boundary consistency, not a full YAML parser. Web properties rendering remains a separate unfinished parity task. No metadata or media files were rewritten.

## Empty properties parity

After web properties presentation was added, comparison found Android's range expression required a content line between delimiters. Empty frontmatter therefore rendered as two rules instead of an omitted empty metadata block. The common range now accepts an immediate exact closing delimiter; extraction handles its empty substring safely.

Tests cover LF/CRLF, optional BOM, closing delimiter at EOF, preserved body and unchanged task ordinals/targeted updates. Both full Android debug suites and module checks passed. This remains pending the next Android batch; the published APK is unchanged.
