package web

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mochil/internal/enrichment"
)

func (m *multiServer) apiMetadataSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	library, ok := m.library(r.URL.Query().Get("library"))
	if !ok {
		http.Error(w, "Library not found", http.StatusNotFound)
		return
	}
	if library.MetadataProvider == "" {
		http.Error(w, "Online metadata is disabled for this shelf", http.StatusConflict)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	switch library.MetadataProvider {
	case "openlibrary":
		if library.Store == nil {
			http.Error(w, "Publication library not found", http.StatusNotFound)
			return
		}
		book, found := library.Store.Book(r.URL.Query().Get("book"))
		if !found {
			http.Error(w, "Book not found", http.StatusNotFound)
			return
		}
		author := firstKnownAuthor(book.Authors)
		results, err := m.openLibrary.Search(ctx, enrichment.Query{Title: book.Title, Author: author})
		if err != nil {
			http.Error(w, "Could not search Open Library: "+err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"provider": "Open Library", "shared": map[string]string{"title": book.Title, "author": author}, "results": results})
	case "tmdb":
		if library.Video == nil {
			http.Error(w, "Video library not found", http.StatusNotFound)
			return
		}
		item, found := library.Video.Item(r.URL.Query().Get("video"))
		if !found {
			http.Error(w, "Video not found", http.StatusNotFound)
			return
		}
		title := item.Title
		if item.Show != "" {
			title = item.Show
		}
		results, err := m.tmdb.Search(ctx, enrichment.TMDBQuery{Title: title, Kind: library.Kind})
		if err != nil {
			http.Error(w, "Could not search TMDB: "+err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"provider": "TMDB", "shared": map[string]string{"title": title}, "results": results})
	case "hardcover":
		if library.Store == nil {
			http.Error(w, "Publication library not found", http.StatusNotFound)
			return
		}
		book, found := library.Store.Book(r.URL.Query().Get("book"))
		if !found {
			http.Error(w, "Book not found", http.StatusNotFound)
			return
		}
		author := firstKnownAuthor(book.Authors)
		results, err := m.hardcover.Search(ctx, enrichment.Query{Title: book.Title, Author: author})
		if err != nil {
			http.Error(w, "Could not search Hardcover: "+err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"provider": "Hardcover", "shared": map[string]string{"title": book.Title, "author": author}, "results": results})
	default:
		http.Error(w, "This provider is configured but its lookup adapter is not available yet", http.StatusNotImplemented)
	}
}

func firstKnownAuthor(authors []string) string {
	for _, author := range authors {
		author = strings.TrimSpace(author)
		if author != "" && !strings.EqualFold(author, "unknown") {
			return author
		}
	}
	return ""
}

func (m *multiServer) metadataSuggestionArtworkHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if m.metadataArtwork == nil {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	path, contentType, err := m.metadataArtwork.Fetch(ctx, r.URL.Query().Get("url"))
	if err != nil {
		http.Error(w, "Artwork is temporarily unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, path)
}

func (m *multiServer) apiPublicationMetadata(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/publication-metadata/"), "/"), "/")
	if len(parts) != 2 || m.metadataOverrides == nil {
		http.NotFound(w, r)
		return
	}
	libraryID, libraryErr := url.PathUnescape(parts[0])
	bookID, bookErr := url.PathUnescape(parts[1])
	library, ok := m.library(libraryID)
	providerName := publicationMetadataProviderName(library.MetadataProvider)
	if libraryErr != nil || bookErr != nil || !ok || library.Store == nil || providerName == "" {
		http.NotFound(w, r)
		return
	}
	if _, exists := library.Store.Book(bookID); !exists {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		value, exists := m.metadataOverrides.Get(libraryID, bookID)
		writeJSON(w, http.StatusOK, map[string]any{"configured": exists && value.ArtworkURL != "", "metadata": value})
	case http.MethodPut:
		var input struct {
			ArtworkURL string `json:"artwork_url"`
			Provider   string `json:"provider"`
			ProviderID string `json:"provider_id"`
		}
		if err := decodeJSONRequest(w, r, 16<<10, &input, true); err != nil || input.Provider != providerName || !enrichment.ValidProviderArtworkURL(input.ArtworkURL) {
			http.Error(w, "Invalid "+providerName+" cover selection", http.StatusBadRequest)
			return
		}
		if err := m.metadataOverrides.SetArtwork(libraryID, bookID, input.ArtworkURL, input.Provider, input.ProviderID); err != nil {
			http.Error(w, "Could not save publication cover", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"saved": true})
	case http.MethodDelete:
		if err := m.metadataOverrides.DeleteArtwork(libraryID, bookID); err != nil {
			http.Error(w, "Could not remove publication cover", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"removed": true})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func publicationMetadataProviderName(configured string) string {
	switch configured {
	case "hardcover":
		return "Hardcover"
	case "openlibrary":
		return "Open Library"
	default:
		return ""
	}
}
