package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"mochil/internal/narration"
	"mochil/internal/narration/booktext"
	"mochil/internal/webpub"
)

// How many sentences are made ahead of the one being played.
const narrationAhead = 4

var (
	narrationClipCache = newNarrationClips(96 << 20)
	// narrationLookahead is per book, so one reader moving through another book
	// cannot cancel the sentences being prepared for this one.
	narrationLookaheadMu sync.Mutex
	narrationLookahead   = map[string]int64{}
)

// nextNarrationLookahead claims the newest generation for one book and returns it.
func nextNarrationLookahead(book string) int64 {
	narrationLookaheadMu.Lock()
	defer narrationLookaheadMu.Unlock()
	narrationLookahead[book]++
	return narrationLookahead[book]
}

// narrationLookaheadCurrent reports whether this generation is still the newest for the book.
func narrationLookaheadCurrent(book string, generation int64) bool {
	narrationLookaheadMu.Lock()
	defer narrationLookaheadMu.Unlock()
	return narrationLookahead[book] == generation
}

// narrationBook serves the text of an EPUB for reading aloud, and the audio of one sentence:
//
//	book/{library}/{book}                           the chapters
//	book/{library}/{book}/chapters/{n}              the sentences of chapter n
//	book/{library}/{book}/chapters/{n}/audio/{i}    sentence i as WAV, in the voice named by ?voice=
func (m *multiServer) narrationBook(w http.ResponseWriter, r *http.Request, rest string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(rest, "book/"), "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	library, err1 := url.PathUnescape(parts[0])
	book, err2 := url.PathUnescape(parts[1])
	if err1 != nil || err2 != nil {
		http.NotFound(w, r)
		return
	}
	id := library + "/" + book
	switch {
	case len(parts) == 2:
		m.narrationChapters(w, id)
	case len(parts) == 4 && parts[2] == "chapters":
		n, err := strconv.Atoi(parts[3])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		m.narrationChapterText(w, id, n)
	case len(parts) == 6 && parts[2] == "chapters" && parts[4] == "audio":
		n, err1 := strconv.Atoi(parts[3])
		i, err2 := strconv.Atoi(parts[5])
		if err1 != nil || err2 != nil {
			http.NotFound(w, r)
			return
		}
		m.narrationSentenceAudio(w, r, id, n, i)
	default:
		http.NotFound(w, r)
	}
}

type narrationChaptersReply struct {
	Title    string             `json:"title"`
	Chapters []booktext.Chapter `json:"chapters"`
}

func (m *multiServer) narrationChapters(w http.ResponseWriter, id string) {
	lib, book, ok := m.findBook(id)
	if !ok || !isEPUB(book) || lib.Store == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	key := derivedKey(lib.ID, book, "narration-chapters")
	body, cached := readiumDerived.get(key)
	if !cached {
		epub, closeEPUB, err := openEPUBFile(lib, book)
		if err != nil {
			http.Error(w, "Could not open this book", http.StatusBadGateway)
			return
		}
		defer closeEPUB()
		manifest, err := webpub.Build(epub.reader, epub.size)
		if err != nil {
			http.Error(w, "This book cannot be read aloud", http.StatusUnprocessableEntity)
			return
		}
		body, _ = json.Marshal(narrationChaptersReply{Title: manifest.Metadata.Title, Chapters: booktext.Chapters(manifest)})
		readiumDerived.put(key, body)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write(body)
}

// narrationSentences returns the sentences of one chapter, from the cache when it can.
func (m *multiServer) narrationSentences(id string, chapter int) ([]booktext.Sentence, int, error) {
	lib, book, ok := m.findBook(id)
	if !ok || !isEPUB(book) || lib.Store == nil {
		return nil, http.StatusNotFound, errors.New("not found")
	}
	key := derivedKey(lib.ID, book, "narration-chapter-"+strconv.Itoa(chapter))
	if body, ok := readiumDerived.get(key); ok {
		var sentences []booktext.Sentence
		if json.Unmarshal(body, &sentences) == nil {
			return sentences, 0, nil
		}
	}
	epub, closeEPUB, err := openEPUBFile(lib, book)
	if err != nil {
		return nil, http.StatusBadGateway, errors.New("Could not open this book")
	}
	defer closeEPUB()
	manifest, err := webpub.Build(epub.reader, epub.size)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, errors.New("This book cannot be read aloud")
	}
	chapters := booktext.Chapters(manifest)
	if chapter < 0 || chapter >= len(chapters) {
		return nil, http.StatusNotFound, errors.New("no such chapter")
	}
	sentences, ok := booktext.ChapterSentences(epub.reader, epub.size, chapters[chapter].Href)
	if !ok {
		sentences = nil // a page made of an image has no text
	}
	if sentences == nil {
		sentences = []booktext.Sentence{}
	}
	body, _ := json.Marshal(sentences)
	readiumDerived.put(key, body)
	return sentences, 0, nil
}

func (m *multiServer) narrationChapterText(w http.ResponseWriter, id string, chapter int) {
	sentences, status, err := m.narrationSentences(id, chapter)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	writeJSON(w, http.StatusOK, map[string]any{"chapter": chapter, "sentences": sentences})
}

func (m *multiServer) narrationSentenceAudio(w http.ResponseWriter, r *http.Request, id string, chapter, index int) {
	if m.narration == nil || !m.narration.Available() {
		http.Error(w, "Narration is not set up on this server.", http.StatusServiceUnavailable)
		return
	}
	key, language, ok := m.narrationVoiceRef(r.URL.Query().Get("voice"))
	if !ok {
		http.Error(w, "Unknown voice.", http.StatusBadRequest)
		return
	}
	sentences, status, err := m.narrationSentences(id, chapter)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	if index < 0 || index >= len(sentences) {
		http.NotFound(w, r)
		return
	}
	revision := "unknown"
	if _, book, found := m.findBook(id); found {
		revision = strconv.FormatInt(book.Size, 10) + "\x00" + strconv.FormatInt(book.Modified.UnixNano(), 10)
	}
	clipKey := func(i int) string {
		// The book's content identity belongs in the key: replacing the file in place
		// keeps the id, and the reader would otherwise hear the old sentences.
		return id + "\x00" + revision + "\x00" + strconv.Itoa(chapter) + "\x00" + strconv.Itoa(i) + "\x00" + key
	}
	speak := func(i int) func(ctxWork context.Context) ([]byte, error) {
		return func(work context.Context) ([]byte, error) {
			audio, err := m.narration.Speak(work, sentences[i].Text, key, language)
			if err != nil {
				return nil, err
			}
			return audio.WAV(), nil
		}
	}
	generation := nextNarrationLookahead(id)
	wav, err := narrationClipCache.get(r.Context(), clipKey(index), speak(index))
	if err == nil {
		// Only once this sentence is made, so the lookahead never delays the one being played.
		go contained("narration look-ahead", func() { m.narrationMakeAhead(id, generation, index, len(sentences), clipKey, speak) })
	}
	if err != nil {
		if errors.Is(err, errNarrationBusy) {
			http.Error(w, "Narration is busy.", http.StatusTooManyRequests)
		} else if errors.Is(err, narration.ErrTooLong) {
			http.Error(w, "That is too long to read in one go.", http.StatusRequestEntityTooLarge)
		} else if r.Context().Err() == nil {
			m.root.log.Warn("narration failed", "error", err)
			http.Error(w, "Narration failed.", http.StatusBadGateway)
		}
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Length", strconv.Itoa(len(wav)))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(wav)
}

// narrationMakeAhead has the next few sentences ready before they are asked for. A request
// for a different place makes the older lookahead stop, so a jump does not leave the
// voice busy with sentences nobody will hear.
func (m *multiServer) narrationMakeAhead(book string, generation int64, from, count int, clipKey func(int) string, speak func(int) func(context.Context) ([]byte, error)) {
	for i := from + 1; i <= from+narrationAhead && i < count; i++ {
		if !narrationLookaheadCurrent(book, generation) {
			return
		}
		if narrationClipCache.has(clipKey(i)) {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		_, err := narrationClipCache.get(ctx, clipKey(i), speak(i))
		cancel()
		if err != nil {
			return
		}
	}
}
