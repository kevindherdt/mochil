package web

import (
	"net/http"
	"strings"

	catalogquery "mochil/internal/query"
)

// koReaderHomeBook is one line of "Continue reading" on the plugin's home page.
type koReaderHomeBook struct {
	Title    string   `json:"title"`
	Authors  []string `json:"authors"`
	Fraction float64  `json:"fraction"`
	// Book says which book this is, so the plugin can open it or fetch it. It is left out for audiobooks and the like.
	Book *koReaderBook `json:"book,omitempty"`
}

// koReaderVault is one vault-kind library (a live Obsidian-style folder of Markdown notes, separate
// from the account's own "Notes & wiki" pages) a device can bundle into readable HTML.
type koReaderVault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// koReaderHome is everything the plugin's home page shows: text and counts only, so it stays small on a slow link.
type koReaderHome struct {
	Account     string             `json:"account"`
	Server      string             `json:"server"`
	Continue    []koReaderHomeBook `json:"continue"`
	Books       int                `json:"books"`
	Libraries   int                `json:"libraries"`
	Series      int                `json:"series"`
	Authors     int                `json:"authors"`
	Collections int                `json:"collections"`
	// Vaults lists vault-kind libraries, kept apart from Libraries/Books: a vault is a folder of Markdown
	// notes (see koReaderVaultHTML), not something publicationKind's browse/shelf pipeline understands.
	Vaults []koReaderVault `json:"vaults,omitempty"`
}

func (m *multiServer) koReaderHomeView(account, server string) koReaderHome {
	home := koReaderHome{Account: account, Server: server, Continue: []koReaderHomeBook{}}
	if m.reading != nil {
		for _, view := range m.readingViews(account, false, "", 5) {
			entry := koReaderHomeBook{Title: view.Title, Authors: view.Authors, Fraction: view.Fraction}
			switch view.LibraryKind {
			case "books", "comics", "manga", "mixed":
				book := m.koReaderBookOf(account, view.LibraryID, view.Book)
				entry.Book = &book
			}
			home.Continue = append(home.Continue, entry)
		}
	}
	if m.query != nil {
		home.Books = m.query.Books(catalogquery.BooksRequest{Limit: 1}).Total
		home.Series = len(m.koReaderGroups("series", "", ""))
		home.Authors = len(m.koReaderGroups("authors", "", ""))
		home.Libraries = len(m.koReaderLibraries(account, ""))
	}
	if m.collections != nil {
		home.Collections = len(m.collections.list(account))
	}
	for _, library := range m.libraries {
		if library.Kind == "vault" && library.Documents != nil {
			home.Vaults = append(home.Vaults, koReaderVault{ID: library.ID, Name: library.Name})
		}
	}
	return home
}

func (m *multiServer) serveKoReaderHome(w http.ResponseWriter, account string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, m.koReaderHomeView(account, strings.TrimRight(m.cfg.PublicURL, "/")))
}
