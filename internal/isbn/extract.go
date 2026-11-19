package isbn

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"mochil/internal/epubread"
)

var ErrUnsupported = errors.New("this format cannot be searched for an ISBN")

// Hints are what is already known about the book, which helps tell its own ISBN from another volume's.
type Hints struct {
	Title  string
	Volume *float64
}

// Candidate is one ISBN found in a book, with the reasons it ranks where it does.
type Candidate struct {
	ISBN    string   `json:"isbn"`
	Score   int      `json:"score"`
	Reasons []string `json:"reasons"`
}

// Result: Own is the book's own ISBN when one stands clearly ahead; empty when nothing was found or the call is too
// close (Ambiguous), in which case the candidates are still listed best first.
type Result struct {
	Own        string      `json:"own,omitempty"`
	Ambiguous  bool        `json:"ambiguous,omitempty"`
	Candidates []Candidate `json:"candidates,omitempty"`
	// Asked is set once Hermes has been asked about this book and could not settle it, so a later run moves on to
	// the next books instead of asking the same ones again.
	Asked bool `json:"asked,omitempty"`
}

// The named signals. Each one is a reason a person can read; the numbers only order them.
const (
	inMetadata    = 100
	copyrightPage = 40
	ebookLabel    = 30
	isbnLabel     = 12
	printLabel    = -10
	confirmed     = 30
	namesVolume   = 40
	repeated      = 10
	titleWord     = 6

	alsoByList       = -55
	otherVolume      = -30
	advertisement    = -25
	minimumToBeOwn   = 40
	clearLead        = 25
	windowRunes      = 160
	maxTitleWords    = 3
	maxRepeats       = 2
	maxDocumentBytes = 4 << 20
)

var (
	candidatePattern = regexp.MustCompile(`[0-9][0-9\-\x{00A0} ]{8,18}[0-9Xx]`)
	printWords       = regexp.MustCompile(`(?i)\b(?:paperback|pbk|hardcover|hardback|print|audiobook|audio)\b`)
	ebookWords       = regexp.MustCompile(`(?i)\b(?:e-?isbn|e-?book|ebook|digital|epub|kindle)\b`)
	isbnWord         = regexp.MustCompile(`(?i)\bisbn(?:-1[03])?\b`)
	alsoByWords      = regexp.MustCompile(`(?i)\b(?:also by|other books|other titles|more from|books by|look for|next volume|coming soon|available now|in the series|more in the series|titles in this series)\b`)
	adWords          = regexp.MustCompile(`(?i)\b(?:advertisement|sample chapter|preview|excerpt)\b`)
	volumeWords      = regexp.MustCompile(`(?i)\b(?:volume|vol\.?|book)\s*0*(\d+(?:\.\d+)?)\b|\(v\.?\s*0*(\d+)\b`) // "Volume 3", "(v3 : pbk)"
	copyrightName    = regexp.MustCompile(`(?i)copyright|colophon|imprint|legal|titlepage|title-page|credits`)
	wordPattern      = regexp.MustCompile(`[a-z0-9]+`)
)

// sightings collects every place one ISBN was seen: the best sighting speaks for it, and each further one adds a little.
type sightings struct {
	best       int
	reasons    []string
	seen       int
	inMetadata bool
}

type finder struct {
	hints  Hints
	found  map[string]*sightings
	titles []string
	// snippets keeps the text around each number's best sighting, and front the text of the copyright pages; they are
	// what a person (or a helper) reads to judge a close call. Only Evidence uses them.
	snippets     map[string]snippet
	front        strings.Builder
	frontRunes   int
	leadingText  strings.Builder // the first pages, a fallback when no page says it is the copyright page
	leadingRunes int
}

type snippet struct {
	text  string
	score int
}

func newFinder(hints Hints) *finder {
	f := &finder{hints: hints, found: map[string]*sightings{}, snippets: map[string]snippet{}}
	for _, word := range wordPattern.FindAllString(strings.ToLower(hints.Title), -1) {
		if len(word) >= 4 {
			f.titles = append(f.titles, word)
		}
	}
	return f
}

func (f *finder) add(isbn string, score int, reasons ...string) {
	entry := f.found[isbn]
	if entry == nil {
		f.found[isbn] = &sightings{best: score, reasons: reasons, seen: 1}
		return
	}
	entry.seen++
	if score > entry.best {
		entry.best, entry.reasons = score, reasons
	}
}

// scan reads one document. A label such as "(ebook)" belongs to the ISBN nearest to it, so on a page that lists several
// editions each number keeps its own label; the wider window is for everything else (lists, volume names, title words).
func (f *finder) scan(text string, copyrightDoc bool) {
	type hit struct {
		isbn       string
		start, end int // rune offsets
	}
	runes := []rune(text)
	var hits []hit
	for _, span := range candidatePattern.FindAllStringIndex(text, -1) {
		raw := text[span[0]:span[1]]
		isbn, ok := normalizeRun(raw)
		if !ok {
			continue
		}
		start := len([]rune(text[:span[0]]))
		hits = append(hits, hit{isbn, start, start + len([]rune(raw))})
	}
	for index, h := range hits {
		window := string(runes[max(0, h.start-windowRunes):min(len(runes), h.end+windowRunes)])
		// Trailing text, up to the next number, is this number's label ("… (ebook)"); only when it says nothing is the
		// text just before it ("E-ISBN: …") used, so a label between two numbers is never read for both.
		after, before := h.end+windowRunes, max(0, h.start-windowRunes/2)
		if index+1 < len(hits) {
			after = min(after, hits[index+1].start)
		}
		if index > 0 {
			before = max(before, hits[index-1].end)
		}
		near := string(runes[h.end:min(len(runes), after)])
		if !ebookWords.MatchString(near) && !printWords.MatchString(near) {
			near += " " + string(runes[before:h.start])
		}
		score := 0
		var reasons []string
		note := func(points int, why string) { score += points; reasons = append(reasons, why) }
		if copyrightDoc {
			note(copyrightPage, "on the copyright page")
		}
		switch {
		case ebookWords.MatchString(near):
			note(ebookLabel, "labelled as the ebook")
		case printWords.MatchString(near):
			note(printLabel, "labelled as a print edition")
		}
		if isbnWord.MatchString(window) {
			note(isbnLabel, "labelled ISBN")
		}
		if alsoByWords.MatchString(window) {
			note(alsoByList, "in an \"also by\" or coming-soon list")
		}
		if adWords.MatchString(window) {
			note(advertisement, "in a preview or advertisement")
		}
		if f.hints.Volume != nil {
			own, other := false, false
			for _, m := range volumeWords.FindAllStringSubmatch(near, -1) {
				if number, err := strconv.ParseFloat(m[1]+m[2], 64); err == nil {
					own = own || number == *f.hints.Volume
					other = other || number != *f.hints.Volume
				}
			}
			switch {
			case own:
				note(namesVolume, "names this volume")
			case other:
				note(otherVolume, "names a different volume")
			}
		}
		matched := 0
		lower := strings.ToLower(window)
		for _, word := range f.titles {
			if matched < maxTitleWords && strings.Contains(lower, word) {
				matched++
			}
		}
		if matched > 0 {
			note(matched*titleWord, "near words from the title")
		}
		f.add(h.isbn, score, reasons...)
		if current, have := f.snippets[h.isbn]; !have || score > current.score {
			f.snippets[h.isbn] = snippet{text: strings.Join(strings.Fields(window), " "), score: score}
		}
	}
}

// keepFront remembers the text of copyright pages (and, as a fallback, of the first pages) up to a limit.
func (f *finder) keepFront(text string, copyrightDoc bool, firstPages bool) {
	take := func(into *strings.Builder, runes *int) {
		room := FrontMatterRunes - *runes
		if room <= 0 {
			return
		}
		cleaned := []rune(strings.Join(strings.Fields(text), " "))
		if len(cleaned) > room {
			cleaned = cleaned[:room]
		}
		into.WriteString(string(cleaned))
		into.WriteString("\n")
		*runes += len(cleaned) + 1
	}
	if copyrightDoc {
		take(&f.front, &f.frontRunes)
	}
	if firstPages {
		take(&f.leadingText, &f.leadingRunes)
	}
}

// normalizeRun picks the ISBN out of a run of digits and spaces: the whole run, or a stretch of it ("2021 978…").
func normalizeRun(raw string) (string, bool) {
	if isbn, ok := Normalize(raw); ok {
		return isbn, true
	}
	groups := strings.Fields(strings.ReplaceAll(raw, " ", " "))
	for length := len(groups) - 1; length >= 1; length-- {
		for from := 0; from+length <= len(groups); from++ {
			if isbn, ok := Normalize(strings.Join(groups[from:from+length], "")); ok {
				return isbn, true
			}
		}
	}
	return "", false
}

func (f *finder) result() Result {
	var out Result
	for isbn, entry := range f.found {
		reasons := append([]string{}, entry.reasons...)
		score := entry.best
		if entry.inMetadata && entry.seen > 1 {
			score += confirmed
			reasons = append(reasons, "the same number is printed in the book")
		}
		if entry.seen > 1 {
			score += repeated * min(entry.seen-1, maxRepeats)
			reasons = append(reasons, "appears "+strconv.Itoa(entry.seen)+" times")
		}
		out.Candidates = append(out.Candidates, Candidate{ISBN: isbn, Score: score, Reasons: reasons})
	}
	sort.Slice(out.Candidates, func(i, j int) bool {
		if out.Candidates[i].Score != out.Candidates[j].Score {
			return out.Candidates[i].Score > out.Candidates[j].Score
		}
		return out.Candidates[i].ISBN < out.Candidates[j].ISBN
	})
	if len(out.Candidates) == 0 {
		return out
	}
	top := out.Candidates[0]
	switch {
	case top.Score < minimumToBeOwn:
		out.Ambiguous = true
	case len(out.Candidates) > 1 && top.Score-out.Candidates[1].Score < clearLead:
		out.Ambiguous = true
	default:
		out.Own = top.ISBN
	}
	return out
}

// Extract reads a book's file for its ISBN. An EPUB is searched through its own metadata and every chapter; a comic
// archive through its ComicInfo.xml.
func Extract(file epubread.ReadSeeker, format string, hints Hints) (Result, error) {
	switch strings.ToLower(format) {
	case "epub":
		return extractEPUB(file, hints)
	case "cbz":
		return extractComic(file)
	}
	return Result{}, ErrUnsupported
}

func extractEPUB(file epubread.ReadSeeker, hints Hints) (Result, error) {
	f, err := searchEPUB(file, hints)
	if err != nil {
		return Result{}, err
	}
	return f.result(), nil
}

func searchEPUB(file epubread.ReadSeeker, hints Hints) (*finder, error) {
	book, err := epubread.Open(file)
	if err != nil {
		return nil, err
	}
	f := newFinder(hints)
	for _, id := range book.Identifiers {
		if isbn, ok := Normalize(strings.TrimPrefix(strings.ToLower(id.Value), "urn:isbn:")); ok {
			f.add(isbn, inMetadata, "listed in the book's own metadata")
			f.found[isbn].inMetadata = true
		}
	}
	for index, document := range book.Spine {
		text, err := book.Text(document, maxDocumentBytes)
		if err != nil {
			continue // one broken chapter should not hide the rest of the book
		}
		copyright := isCopyrightDocument(document, text)
		f.scan(text.Body, copyright)
		f.keepFront(text.Body, copyright, index < 3)
	}
	return f, nil
}

func isCopyrightDocument(document epubread.Document, text epubread.Text) bool {
	switch document.Guide {
	case "copyright-page", "colophon", "title-page":
		return true
	}
	for _, label := range text.Types {
		if label == "copyright-page" || label == "colophon" || label == "imprint" {
			return true
		}
	}
	return copyrightName.MatchString(document.Name)
}

// extractComic reads the GTIN a ComicInfo.xml may carry.
func extractComic(file epubread.ReadSeeker) (Result, error) {
	archive, err := epubread.ZIP(file)
	if err != nil {
		return Result{}, err
	}
	f := newFinder(Hints{})
	for _, entry := range archive.File {
		if !strings.EqualFold(entry.Name, "ComicInfo.xml") {
			continue
		}
		if gtin, ok := comicGTIN(entry); ok {
			if isbn, valid := Normalize(gtin); valid {
				f.add(isbn, inMetadata, "listed in the comic's ComicInfo.xml")
			}
		}
	}
	return f.result(), nil
}

func comicGTIN(entry *zip.File) (string, bool) {
	reader, err := entry.Open()
	if err != nil {
		return "", false
	}
	defer reader.Close()
	var info struct {
		GTIN string `xml:"GTIN"`
	}
	if err := xml.NewDecoder(io.LimitReader(reader, 4<<20)).Decode(&info); err != nil {
		return "", false
	}
	return info.GTIN, strings.TrimSpace(info.GTIN) != ""
}
