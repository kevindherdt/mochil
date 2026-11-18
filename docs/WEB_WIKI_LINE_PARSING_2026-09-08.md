# Sequential Markdown line parsing

The shared web Markdown renderer previously normalized the full document into another string and split it into an array of every line before processing blocks. `markdown-lines.js` now provides a sequential cursor using CRLF/lone-CR/LF delimiters and one-line lookahead. The renderer consumes it without the normalized-document copy and eager line array.

Existing heading, list, quote, code-fence, rule, paragraph, inline escaping, and trusted vault rendering callbacks remain unchanged. This is not streaming output or DOM virtualization: the original source, block buffers, and final HTML are still retained as needed. No device/browser latency or memory percentage is claimed.

Tests compare cursor output against the previous normalized split for 512 fragment combinations, including Unicode and trailing empty lines. Explicit renderer regressions cover adjacent block boundaries, mixed endings, unterminated fences, escaping, and empty input. All 502 frontend tests, module checks, and isolated vault-mutation browser checks at 1440px and 390px passed.

No layout changes or deployment. This remains in the batch after 0.16.77.

## Link normalization guard

The safe-link allowlist rejected protocol-relative `//host` destinations but accepted slash-backslash authority forms that browsers normalize to external hosts. Destinations with backslashes or ASCII controls are now rejected before the allowlist. Explicit HTTP(S), mailto, ordinary local paths, and fragments remain supported. Rejected links render as escaped text; this is navigation hardening, not a claim of a demonstrated script-execution exploit.

Tests verify actual URL normalization of slash-backslash authority, rejection of malformed/control-containing destinations, and preservation of supported links. All 505 frontend tests, module checks, and vault-mutation browser fixtures passed. No deployment yet.
