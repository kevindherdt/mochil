package readpos

import (
	"errors"
	"fmt"
	"io"
)

// Quality says how far a converted position can be trusted.
type Quality int

const (
	// QualityChapter means only the chapter is known.
	QualityChapter Quality = iota + 1
	// QualityApproximate means the structure partly resolved; the text is the
	// start of the nearest element that did.
	QualityApproximate
	// QualityText means the position resolved to a specific place in the text.
	QualityText
)

var (
	ErrNotFound = errors.New("position text was not found in the publication")
	ErrInvalid  = errors.New("invalid position")
)

// acceptableScore is the lowest score treated as a confident text match: a hit
// in the preferred chapter or with matching context. Matches below it are
// repeated-sentence guesses in other chapters.
const confidentScore = 150

// Book answers position questions about one EPUB. It is not safe for concurrent
// use; create one per operation.
type Book struct {
	spine    []spineItem
	total    int64
	before   []int64
	chapters map[int]*chapterText
}

func Open(reader io.ReaderAt, size int64) (*Book, error) {
	spine, err := openSpine(reader, size)
	if err != nil {
		return nil, err
	}
	book := &Book{spine: spine, before: make([]int64, len(spine)), chapters: map[int]*chapterText{}}
	for i, item := range spine {
		book.before[i] = book.total
		book.total += max(item.size, 1)
	}
	return book, nil
}

func (b *Book) chapter(index int) (*chapterText, error) {
	if index < 0 || index >= len(b.spine) {
		return nil, ErrInvalid
	}
	if cached, ok := b.chapters[index]; ok {
		return cached, nil
	}
	data, err := readZip(b.spine[index].file, maximumChapterBytes)
	if err != nil {
		return nil, err
	}
	body, err := parseChapter(data)
	if err != nil {
		return nil, fmt.Errorf("spine item %d: %w", index, err)
	}
	if len(b.chapters) >= 6 {
		clear(b.chapters)
	}
	chapter := newChapterText(body)
	b.chapters[index] = chapter
	return chapter, nil
}

// bookFraction approximates reading progress from uncompressed chapter sizes.
func (b *Book) bookFraction(index int, within float64) float64 {
	return (float64(b.before[index]) + float64(max(b.spine[index].size, 1))*within) / float64(b.total)
}

// AnchorFromXPointer converts a KOReader position to a web text anchor. When the
// position cannot be tied to chapter text the anchor carries only the chapter.
func (b *Book) AnchorFromXPointer(value string) (Anchor, Quality, error) {
	xp, ok := parseXPointer(value)
	if !ok || xp.fragment > len(b.spine) {
		return Anchor{}, 0, ErrInvalid
	}
	index := xp.fragment - 1
	chapterOnly := Anchor{SpineIndex: index, Progress: b.bookFraction(index, 0)}
	chapter, err := b.chapter(index)
	if err != nil {
		return chapterOnly, QualityChapter, err
	}
	flat, level := chapter.resolveXPointer(xp)
	if level == qualityChapter || len(chapter.flat) == 0 {
		return chapterOnly, QualityChapter, nil
	}
	prefix, exact, suffix, ok := chapter.anchorAt(flat)
	if !ok {
		return chapterOnly, QualityChapter, nil
	}
	within := float64(chapter.squashedAtFlat(flat)) / float64(max(len(chapter.squashed), 1))
	anchor := Anchor{SpineIndex: index, Prefix: prefix, Exact: exact, Suffix: suffix, Progress: b.bookFraction(index, within)}
	if level == qualityApproximate {
		return anchor, QualityApproximate, nil
	}
	return anchor, QualityText, nil
}

// XPointerFromAnchor finds the anchor's quoted text and returns the KOReader
// position of its first word. Chapters near the anchor's own are tried first.
func (b *Book) XPointerFromAnchor(a Anchor) (string, Quality, error) {
	if a.SpineIndex < 0 || a.SpineIndex >= len(b.spine) {
		return "", 0, ErrInvalid
	}
	order := []int{a.SpineIndex, a.SpineIndex - 1, a.SpineIndex + 1}
	for index := range b.spine {
		if index < a.SpineIndex-1 || index > a.SpineIndex+1 {
			order = append(order, index)
		}
	}
	var best match
	bestChapter, found := 0, false
	for _, index := range order {
		if index < 0 || index >= len(b.spine) {
			continue
		}
		chapter, err := b.chapter(index)
		if err != nil {
			continue
		}
		candidate, ok := chapter.findAnchor(a, index == a.SpineIndex, func(sq int) float64 {
			return b.bookFraction(index, float64(sq)/float64(max(len(chapter.squashed), 1)))
		})
		if ok && (!found || candidate.score > best.score) {
			best, bestChapter, found = candidate, index, true
		}
		if found && best.score >= 215 {
			break
		}
	}
	if !found || best.score < confidentScore && bestChapter != a.SpineIndex {
		return "", 0, ErrNotFound
	}
	chapter, err := b.chapter(bestChapter)
	if err != nil {
		return "", 0, err
	}
	xpointer, err := chapter.buildXPointer(bestChapter+1, best.flat)
	if err != nil {
		return "", 0, err
	}
	return xpointer, QualityText, nil
}
