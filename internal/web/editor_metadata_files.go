package web

// editor_metadata_files.go — writing the metadata Mochil knows back out beside the book.
//
// Everything the editor does by default lives in Mochil's own override store. That is right for Mochil and invisible to
// every other program, which is why a folder of books whose titles, series and ISBNs were fixed here still arrives in
// Calibre carrying whatever the EPUB was born with — Kevin's actual workflow is "fix it in Mochil, then drag the folders
// into Calibre". Calibre reads a metadata.opf beside the book when it adds one, so the bridge is to write that sidecar
// from the fields as they are now (override included), and optionally to write them into the EPUB's own package as well.
//
// Nothing here is a second writer to anything: it writes only inside a library Mochil owns, through the same executor
// that already imports and embeds for the managed library, and it refuses Calibre libraries outright — Calibre keeps its
// own metadata.opf and would overwrite ours on its next save.

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mochil/internal/catalog"
	"mochil/internal/ingest"
	"mochil/internal/isbn"
	"mochil/internal/opf"
)

// editorMaximumSidecars matches the batch size the rest of the editor uses for a single write.
const editorMaximumSidecars = 100

const calibreOwnsSidecarMessage = "This library's files belong to Calibre, which keeps its own metadata.opf beside every book. " +
	"Writing ours there would fight it, so this stays Calibre's job. Fix the metadata in a library Mochil owns and add those folders to Calibre, or import the folders into Calibre itself."

// sidecarMetadata maps a catalog book onto what the sidecar writer takes. Rating is already 0-10 here (two per star),
// which is exactly what Calibre stores, so it is passed through rather than converted a second time.
func sidecarMetadata(book catalog.Book) opf.Metadata {
	return opf.Metadata{
		Title:       book.Title,
		TitleSort:   book.TitleSort,
		Authors:     book.Authors,
		AuthorSort:  strings.Join(book.AuthorSort, " & "),
		Series:      book.Series,
		SeriesIndex: book.SeriesIndex,
		Tags:        book.Tags,
		Publisher:   book.Publisher,
		Published:   book.Published,
		Language:    book.Language,
		Rating:      book.Rating,
		Description: book.Description,
		Identifiers: book.Identifiers,
	}
}

type metadataFileResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"` // written, exists, failed
	Sidecar string `json:"sidecar,omitempty"`
	Message string `json:"message,omitempty"`
}

func (m *multiServer) editorMetadataFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Library   string   `json:"library"`
		IDs       []string `json:"ids"`
		Sidecar   bool     `json:"sidecar"`
		Embed     bool     `json:"embed"`
		Overwrite bool     `json:"overwrite"`
	}
	if err := decodeJSONRequest(w, r, editorMaximumBody, &req, true); err != nil {
		http.Error(w, "The request could not be read.", http.StatusBadRequest)
		return
	}
	lib, ok := m.editorLibrary(w, req.Library)
	if !ok {
		return
	}
	if calibreOwnsMetadata(lib) {
		http.Error(w, calibreOwnsSidecarMessage, http.StatusConflict)
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > editorMaximumSidecars {
		http.Error(w, fmt.Sprintf("Write metadata for between 1 and %d books at a time.", editorMaximumSidecars), http.StatusBadRequest)
		return
	}
	// A sidecar is the default because it never touches the book file; embedding is asked for.
	sidecar := req.Sidecar || !req.Embed
	// A LibraryWriter rather than the ingest executor: a folder of books dragged in — which is often the library a
	// person is fixing — is exactly the case the executor refuses as a copy target, and a sidecar is not a copy.
	writer, err := ingest.NewLibraryWriter(lib.SourcePath)
	if err != nil || !publicationKind(lib.Kind) {
		http.Error(w, "This library is not a folder Mochil can write to.", http.StatusConflict)
		return
	}
	results := make([]metadataFileResult, 0, len(req.IDs))
	rekeyed := false
	for _, id := range req.IDs {
		book, found := lib.Store.Book(id)
		if !found {
			results = append(results, metadataFileResult{ID: id, Status: "failed", Message: "That book is not in this library."})
			continue
		}
		result := metadataFileResult{ID: id, Title: book.Title}
		// Embedding rewrites the book, and every finding about a book is keyed on its size and modification time — so
		// the match would disappear at the moment it was used, and a staged Replace would have nothing left to apply.
		// The finding is still true; it is the file that moved, so it is filed under the file as it is afterwards.
		var matchFinding bulkMatch
		hadMatch := false
		var isbnFinding isbn.Result
		hadISBN := false
		if req.Embed {
			if m.matchFound != nil {
				matchFinding, hadMatch = m.matchFound.Get(lib.ID, id, book.Size, book.Modified)
			}
			if m.isbnFound != nil {
				isbnFinding, hadISBN = m.isbnFound.Get(lib.ID, id, book.Size, book.Modified)
			}
		}
		meta := sidecarMetadata(book)
		if strings.TrimSpace(meta.Title) == "" {
			result.Status, result.Message = "failed", "A book without a title cannot be written."
			results = append(results, result)
			continue
		}
		if sidecar {
			written, err := writer.WriteSidecar(book.Path, meta, req.Overwrite)
			switch {
			case err == nil:
				result.Status, result.Sidecar = "written", written
			case errors.Is(err, opf.ErrExists):
				result.Status, result.Sidecar = "exists", written
				result.Message = "A metadata.opf is already there and was left alone."
			default:
				result.Status, result.Message = "failed", err.Error()
			}
		}
		wroteSidecar := result.Status == "written" || result.Status == "exists"
		if req.Embed {
			if err := writer.EmbedMetadata(book.Path, meta); err != nil {
				// A book whose sidecar was written is still reported as failed: embedding was asked for and did not
				// happen, and saying "written" would hide that.
				if wroteSidecar {
					result.Message = "The metadata.opf was written, but the EPUB itself was not: " + err.Error()
				} else {
					result.Message = err.Error()
				}
				result.Status = "failed"
			} else if result.Status != "failed" {
				result.Status = "written"
				if wroteSidecar {
					result.Message = "Written beside the book and into the EPUB."
				} else {
					result.Message = "Written into the EPUB."
				}
			}
		}
		if req.Embed && result.Status == "written" {
			if info, statErr := os.Stat(filepath.Join(lib.SourcePath, filepath.FromSlash(book.Path))); statErr == nil {
				if hadMatch && m.matchFound != nil {
					m.matchFound.Put(lib.ID, id, info.Size(), info.ModTime(), matchFinding)
					rekeyed = true
				}
				if hadISBN && m.isbnFound != nil {
					m.isbnFound.Put(lib.ID, id, info.Size(), info.ModTime(), isbnFinding)
					rekeyed = true
				}
			}
		}
		results = append(results, result)
	}
	if rekeyed {
		if m.matchFound != nil {
			_ = m.matchFound.Save()
		}
		if m.isbnFound != nil {
			_ = m.isbnFound.Save()
		}
	}
	summary := map[string]int{"written": 0, "exists": 0, "failed": 0}
	for _, result := range results {
		summary[result.Status]++
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "summary": summary})
}
