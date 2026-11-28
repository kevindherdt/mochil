package web

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"mochil/internal/reading"
)

type readingPosition = reading.Position
type webAnchor = reading.Anchor

var errReadingConflict = reading.ErrConflict

func readingProgress(value koSyncProgress) reading.SyncProgress {
	return reading.SyncProgress{Document: value.Document, Progress: value.Progress, Percentage: value.Percentage, Device: value.Device, DeviceID: value.DeviceID, Timestamp: value.Timestamp, SeenAt: value.SeenAt}
}

func validWebAnchor(anchor *webAnchor) bool {
	return reading.ValidAnchor(anchor)
}

type readingView struct {
	bookView
	CFI          string  `json:"cfi,omitempty"`
	Fraction     float64 `json:"fraction"`
	UpdatedAt    int64   `json:"updated_at"`
	Revision     int64   `json:"revision"`
	DeviceID     string  `json:"device_id,omitempty"`
	Finished     bool    `json:"finished,omitempty"`
	Status       string  `json:"status,omitempty"`
	FinishedAt   int64   `json:"finished_at,omitempty"`
	KoSyncSeenAt int64   `json:"kosync_seen_at,omitempty"`
}

// recordKoSyncProgress mirrors standard KoSync updates into the dashboard.
// A precision-plugin link is authoritative; filename matching is accepted only
// when it identifies exactly one publication across all configured shelves.
func (m *multiServer) recordKoSyncProgress(username string, progress koSyncProgress) {
	m.recordKoSyncProgressAt(username, progress, 0)
}

// recordKoSyncProgressAt mirrors an update that a digest pass just replayed,
// remembering which table generation it was tried against.
func (m *multiServer) recordKoSyncProgressAt(username string, progress koSyncProgress, attempted uint64) {
	if m.reading == nil {
		return
	}
	account, ok := m.koSyncAccount(username)
	if !ok {
		return
	}
	libraryID, bookID, ok := m.koSyncBook(account, progress.Document)
	if !ok {
		m.noteUnmatchedKoSyncAt(username, progress, attempted)
		return
	}
	_, _ = m.readingSync.Record(account, libraryID, bookID, readingProgress(progress), m.anchorForKoReader(libraryID, bookID, progress.Progress))
}

func (m *multiServer) koSyncAccount(username string) (string, bool) {
	account := m.readingAccount(nil)
	if m.accountStore() != nil {
		identity, ok := m.accountStore().IdentityByUsername(username)
		if !ok {
			return "", false
		}
		account = identity.ID
	}
	return account, true
}

// koSyncBook finds the publication a KoSync document hash refers to. A link
// from the precision plugin is authoritative; filename matching is accepted
// only when it identifies exactly one publication across all shelves.
func (m *multiServer) koSyncBook(account, document string) (libraryID, bookID string, ok bool) {
	if linked, found := m.readingSync.FindDocument(account, document); found {
		return linked.LibraryID, linked.BookID, true
	}
	matches := 0
	for _, library := range m.libraries {
		if library.Store == nil {
			continue
		}
		for _, book := range library.Store.Books() {
			if koSyncFilenameMD5(path.Base(book.Path)) == document {
				if matches++; matches > 1 {
					return "", "", false
				}
				libraryID, bookID = library.ID, book.ID
			}
		}
	}
	if matches == 0 {
		// KOReader's default document id is a hash of the file's content.
		if found := m.koSyncDigestMatches(document); len(found) == 1 {
			return found[0][0], found[0][1], true
		}
	}
	return libraryID, bookID, matches == 1
}

func (m *multiServer) readingView(position readingPosition) (readingView, bool) {
	lib, book, ok := m.findBook(position.LibraryID + "/" + position.BookID)
	if !ok {
		return readingView{}, false
	}
	return readingView{bookView: m.bookView(lib, book), CFI: position.CFI, Fraction: position.Fraction, UpdatedAt: position.UpdatedAt, Revision: position.Revision, DeviceID: position.DeviceID, Finished: position.Finished, Status: position.Status, FinishedAt: position.FinishedAt, KoSyncSeenAt: position.KoSyncSeenAt}, true
}

func (m *multiServer) apiReading(w http.ResponseWriter, r *http.Request) {
	if m.reading == nil || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("held") == "true" {
		limit := 0
		_, _ = fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)
		writeJSON(w, http.StatusOK, m.heldReadingViews(m.readingAccount(r), limit))
		return
	}
	finished := r.URL.Query().Get("finished") == "true"
	limit := 12
	if value := r.URL.Query().Get("limit"); value != "" {
		_, _ = fmt.Sscanf(value, "%d", &limit)
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	libraryID := strings.TrimSpace(r.URL.Query().Get("library"))
	writeJSON(w, http.StatusOK, m.readingViews(m.readingAccount(r), finished, libraryID, limit))
}

func (m *multiServer) readingViews(account string, finished bool, libraryID string, limit int) []readingView {
	// Positions whose book is no longer in the catalogue are dropped below, so a
	// fixed window can come back short (or empty) while readable rows wait further
	// down the history. Grow the window until the page is full or the history ends.
	fetch := limit
	for {
		// The in-progress shelf uses the same rule as the device: actually read (not opened on
		// page one), and not parked as on hold or dropped. Finished books keep the plain list.
		var positions []reading.Position
		if finished {
			positions = m.reading.ListLibraryLimit(account, true, libraryID, fetch)
		} else {
			positions = m.reading.ListContinue(account, libraryID, fetch)
		}
		out := make([]readingView, 0, min(limit, len(positions)))
		for _, position := range positions {
			if view, ok := m.readingView(position); ok {
				out = append(out, view)
			}
			if len(out) == limit {
				return out
			}
		}
		if fetch <= 0 || len(positions) < fetch {
			return out
		}
		fetch *= 2
		if fetch > 4096 {
			fetch = 0 // unbounded: take the rest in one pass
		}
	}
}

func (m *multiServer) apiReadingBook(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/api/reading/")
	action := ""
	for _, suffix := range []string{"/finish", "/restore", "/dismiss", "/show", "/status", "/kosync-seen", "/kosync-link", "/history"} {
		if strings.HasSuffix(remainder, suffix) {
			action, remainder = strings.TrimPrefix(suffix, "/"), strings.TrimSuffix(remainder, suffix)
			break
		}
	}
	lib, book, ok := m.findBook(remainder)
	if !ok || m.reading == nil {
		http.NotFound(w, r)
		return
	}
	account := m.readingAccount(r)
	if action != "" {
		if action == "status" {
			m.apiReadingStatus(w, r, lib, book, account)
			return
		}
		if action == "history" {
			switch r.Method {
			case http.MethodGet:
				writeJSON(w, http.StatusOK, m.reading.History(account, lib.ID, book.ID))
			case http.MethodPost:
				var request struct {
					UpdatedAt int64 `json:"updated_at"`
				}
				if decodeJSONRequest(w, r, 4096, &request, false) != nil || request.UpdatedAt <= 0 {
					http.Error(w, "Invalid history position", http.StatusBadRequest)
					return
				}
				position, err := m.reading.RestoreHistory(account, lib.ID, book.ID, request.UpdatedAt)
				if errors.Is(err, os.ErrNotExist) {
					http.NotFound(w, r)
					return
				}
				if err != nil {
					http.Error(w, "Reading history could not be restored", http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, position)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if action == "kosync-seen" {
			var request struct {
				Timestamp int64      `json:"timestamp"`
				CFI       string     `json:"cfi"`
				Fraction  *float64   `json:"fraction"`
				WebAnchor *webAnchor `json:"web_anchor"`
			}
			if decodeJSONRequest(w, r, 32<<10, &request, false) != nil || request.Timestamp <= 0 || len(request.CFI) > reading.MaximumCFIBytes || (request.Fraction != nil && !reading.ValidFraction(*request.Fraction)) || !validWebAnchor(request.WebAnchor) {
				http.Error(w, "Invalid KoSync timestamp", http.StatusBadRequest)
				return
			}
			var position readingPosition
			var err error
			if request.Fraction != nil {
				position, err = m.readingSync.Accept(account, lib.ID, book.ID, request.Timestamp, readingPosition{CFI: request.CFI, Fraction: *request.Fraction, WebAnchor: request.WebAnchor})
			} else {
				position, err = m.readingSync.MarkSeen(account, lib.ID, book.ID, request.Timestamp)
			}
			if err != nil {
				http.Error(w, "KoSync decision could not be saved", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, position)
			return
		}
		if action == "kosync-link" {
			var request struct {
				Document string `json:"document"`
			}
			if decodeJSONRequest(w, r, 4096, &request, false) != nil || !koSyncDocumentID.MatchString(request.Document) {
				http.Error(w, "Invalid KoSync document", http.StatusBadRequest)
				return
			}
			position, err := m.readingSync.Link(account, lib.ID, book.ID, request.Document)
			if err != nil {
				http.Error(w, "KoSync link could not be saved", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, position)
			return
		}
		if action == "dismiss" || action == "show" {
			position, err := m.reading.SetDismissed(account, lib.ID, book.ID, action == "dismiss")
			if errors.Is(err, os.ErrNotExist) {
				http.NotFound(w, r)
				return
			}
			if err != nil {
				http.Error(w, "Dashboard state could not be saved", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, position)
			return
		}
		position, err := m.reading.SetFinished(account, lib.ID, book.ID, action == "finish")
		if err != nil {
			http.Error(w, "Reading state could not be saved", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, position)
		return
	}
	switch r.Method {
	case http.MethodGet:
		position, exists := m.reading.Get(account, lib.ID, book.ID)
		if !exists {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, position)
	case http.MethodPut:
		var request struct {
			CFI             string            `json:"cfi"`
			Fraction        float64           `json:"fraction"`
			DeviceID        string            `json:"device_id"`
			DeviceName      string            `json:"device_name"`
			BaseRevision    int64             `json:"base_revision"`
			WebAnchor       *webAnchor        `json:"web_anchor"`
			Locators        map[string]string `json:"locators"`
			KoReaderLocator string            `json:"koreader_locator"`
		}
		if err := decodeJSONRequest(w, r, 64<<10, &request, false); err != nil || !reading.ValidFraction(request.Fraction) || len(request.CFI) > reading.MaximumCFIBytes || len(request.DeviceID) > reading.MaximumDeviceIDBytes || len(request.DeviceName) > reading.MaximumDeviceNameBytes || !validWebAnchor(request.WebAnchor) || !reading.ValidLocators(request.Locators) || !reading.ValidKoReaderLocator(request.KoReaderLocator) {
			http.Error(w, "Invalid reading position", http.StatusBadRequest)
			return
		}
		// Android reports a CREngine-style location but often no usable text quote.
		// Derive the quote so the web reader and KOReader can open the same page.
		if (request.WebAnchor == nil || request.WebAnchor.Exact == "") && request.KoReaderLocator != "" {
			if derived := m.anchorForKoReader(lib.ID, book.ID, request.KoReaderLocator); derived != nil {
				request.WebAnchor = derived
			}
		}
		position, err := m.reading.Put(account, readingPosition{LibraryID: lib.ID, BookID: book.ID, CFI: request.CFI, Fraction: request.Fraction, DeviceID: request.DeviceID, DeviceName: request.DeviceName, WebAnchor: request.WebAnchor, Locators: request.Locators, KoReaderLocator: request.KoReaderLocator}, request.BaseRevision)
		if errors.Is(err, errReadingConflict) {
			writeJSON(w, http.StatusConflict, position)
			return
		}
		if err != nil {
			http.Error(w, "Reading position could not be saved", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, position)
	case http.MethodDelete:
		if err := m.reading.Delete(account, lib.ID, book.ID); err != nil {
			http.Error(w, "Reading state could not be removed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
