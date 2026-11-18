# KOReader ↔ web position conversion

Status: implemented 2026-10-26 in `internal/readpos`, wired through
`internal/web/position_bridge.go`. Not yet validated on a physical device.

## Problem

Stock KOReader (no Mochil plugin) syncs a crengine XPointer plus a
percentage. The web reader stores a text anchor (spine index, quoted words,
context). Before this change the browser guessed at an XPointer's DOM node, and
a position saved in the web reader never reached stock KOReader at all.

## Approach

Structure proposes, text decides.

- `AnchorFromXPointer`: resolve the XPointer in the chapter (element steps count
  same-named siblings, `text()` steps count addressable text nodes, offsets are
  in whitespace-collapsed engine text), snap to a word start, and read the
  surrounding words as an anchor. Partial structure degrades to the nearest
  element (`QualityApproximate`); no structure degrades to the chapter.
- `XPointerFromAnchor`: find the quoted text with all whitespace removed,
  scoring repeated passages by chapter preference, context match and distance
  from the reader's book fraction, then serialise the XPointer. A text that is
  not found returns `ErrNotFound`; weak matches in other chapters are rejected.
  Nothing is guessed.

Matching ignores whitespace entirely, so minified XHTML, pretty-printed XHTML
and crengine's text all agree. The web reader's resolver uses the same rule.

## Wiring

- KoSync `PUT /syncs/progress` and the plugin progress route store a computed
  anchor when the client did not send one.
- KoSync `GET /syncs/progress/{doc}` returns the web position, as an XPointer,
  when it is newer than KOReader's stored progress and converts at
  `QualityText`; otherwise the stored progress is returned unchanged.

## Verification

- Unit tests for round trips, inline elements, entities, minified chapters,
  repeated sentences, missing text and malformed pointers.
- `READPOS_CORPUS=/path/to/epubs go test ./internal/readpos -run Corpus -v`
  round-trips sampled positions in real books (447 samples across 40 books: no
  failures or unstable results).
- An HTTP test drives stock KOReader → web anchor → web save → KOReader.

## Known limits

- The crengine text model is reproduced from documented behaviour. Offsets may
  differ from the engine by a few characters inside a text node; structure
  (which paragraph) should not. Validate against a real device or the KOReader
  emulator before relying on character-exact placement.
- Kobo's native sync (`koboSpan` ids) is not handled here.
- DocFragment N is taken to be spine item N, as the plugin already assumes.
