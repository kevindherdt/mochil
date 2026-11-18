# Web wiki properties presentation

Pending after 0.16.78; no live deployment.

File-backed wiki Markdown now separates an exact leading YAML delimiter block from the note body. Nonempty properties appear in a native, collapsed `details` section labeled Properties. Content is escaped verbatim, including nested YAML; no YAML objects, tags or scripts are evaluated, and no source file is rewritten. Empty blocks create no empty control. Unclosed/non-leading delimiter blocks remain ordinary Markdown.

The dedicated splitter accepts BOM and LF/CRLF/CR while preserving original property text and body. The shared non-vault Markdown renderer is unaffected. Existing code-copy binding can operate on the property code block.

Frontend tests cover delimiter boundaries, escaping, nested text, empty properties and newline preservation. Chromium fixtures at 1440px and 390px passed keyboard toggle, initially collapsed state, 44px summary target and no page overflow with a 600-character property value. These are focused fixtures, not a full application visual review. Impeccable hardening guided native disclosure, keyboard focus and overflow behavior while retaining incumbent theme tokens and page layout.

This is read-only raw-property presentation, not a typed YAML property editor or full Obsidian feature parity. Broader wiki editing/history tasks remain outstanding.
