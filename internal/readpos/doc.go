// Package readpos converts reading positions between KOReader's crengine
// XPointers and the text anchors the web reader stores.
//
// Structure proposes a location and text decides whether it is trustworthy: an
// anchor is only produced for a position that resolves to real chapter text,
// and an XPointer is only produced from an anchor whose quoted text is found in
// the chapter. Anything weaker degrades to a chapter-level result instead of a
// guess, so a bad conversion can never move a good position.
//
// The crengine text model (whitespace collapsing, which text nodes are
// addressable) is reproduced from its documented behaviour and is not verified
// against a physical device; offsets can differ from the engine by a few
// characters inside a text node, never by structure.
package readpos
