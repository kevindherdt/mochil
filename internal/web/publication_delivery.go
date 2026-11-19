package web

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"mochil/internal/acquisition"
	"mochil/internal/artwork"
	"mochil/internal/catalog"
)

func (m *multiServer) readBook(w http.ResponseWriter, r *http.Request) {
	lib, book, ok := m.findBook(strings.TrimPrefix(r.URL.Path, "/read/"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", book.MediaType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	if m.publications != nil {
		if cached, info, hit := m.publications.Open(lib.ID, book); hit {
			defer cached.Close()
			http.ServeContent(w, r, path.Base(book.Path), info.ModTime(), cached)
			return
		}
	}
	opened, err := (acquisition.Service{}).Open(r.Context(), lib.Store, book, r.Header.Get("Range"))
	if err != nil {
		http.Error(w, "Book is temporarily unavailable", http.StatusBadGateway)
		return
	}
	defer opened.Close()
	if opened.Reader != nil {
		http.ServeContent(w, r, path.Base(book.Path), book.Modified, opened.Reader)
		return
	}
	response := opened.Response
	for _, header := range []string{"Content-Length", "Content-Range", "Last-Modified", "ETag"} {
		if value := response.Header.Get(header); value != "" {
			w.Header().Set(header, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	if response.StatusCode == http.StatusOK && r.Header.Get("Range") == "" && m.publications != nil {
		if temporary, target, cacheable := m.publications.Begin(lib.ID, book); cacheable {
			_, copyErr := io.Copy(io.MultiWriter(w, temporary), response.Body)
			if copyErr == nil {
				if m.publications.Commit(temporary, target) {
					m.schedulePublicationMaintenance()
				}
			} else {
				m.publications.Discard(temporary)
			}
			return
		}
	}
	_, _ = io.Copy(w, response.Body)
}

func (m *multiServer) scheduleCoverMaintenance() {
	if m.covers == nil || m.backgroundJobs == nil {
		return
	}
	_ = m.backgroundJobs.Submit("cache:covers:maintain", func(ctx context.Context) error {
		m.covers.maintain(ctx)
		return ctx.Err()
	})
}

func (m *multiServer) schedulePublicationMaintenance() {
	if m.publications == nil || m.backgroundJobs == nil {
		return
	}
	_ = m.backgroundJobs.Submit("cache:publications:maintain", func(ctx context.Context) error {
		m.publications.MaintainContext(ctx)
		return ctx.Err()
	})
}

func (m *multiServer) cover(w http.ResponseWriter, r *http.Request) {
	lib, book, ok := m.findBook(strings.TrimPrefix(r.URL.Path, "/covers/"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	cacheID := book.ID
	providerArtwork := ""
	if m.metadataOverrides != nil {
		if value, exists := m.metadataOverrides.Get(lib.ID, book.ID); exists {
			providerArtwork = value.ArtworkURL
			if providerArtwork != "" {
				digest := sha256.Sum256([]byte(providerArtwork))
				cacheID += "-provider-" + fmt.Sprintf("%x", digest[:6])
			}
		}
	}
	if modified := coverModified(book); !modified.IsZero() {
		cacheID += "-" + strconv.FormatInt(modified.Unix(), 36)
	}
	// Grids ask for a width that fits the card; "full" is the stored cover.
	variantWidth := coverVariantWidth(r.URL.Query().Get("w"))
	variantID := cacheID + "-w" + strconv.Itoa(variantWidth)
	sendCover := func(full []byte) {
		if variantWidth > 0 && m.covers != nil {
			if small, ok := m.covers.thumbnail(lib.ID, variantID, full, variantWidth); ok {
				full = small
			}
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, max-age=604800, immutable")
		_, _ = w.Write(full)
	}
	if m.covers != nil {
		if variantWidth > 0 {
			if cached, ok := m.covers.read(lib.ID, variantID); ok {
				w.Header().Set("Content-Type", "image/jpeg")
				w.Header().Set("Cache-Control", "private, max-age=604800, immutable")
				_, _ = w.Write(cached)
				return
			}
		}
		if cached, ok := m.covers.read(lib.ID, cacheID); ok {
			sendCover(cached)
			return
		}
	}
	// A nil channel blocks forever, which would hang any server (or test) that never
	// configured a slot limit. No slots configured means no limit.
	if m.coverSlots != nil {
		select {
		case m.coverSlots <- struct{}{}:
			defer func() { <-m.coverSlots }()
		case <-r.Context().Done():
			return
		}
	}
	// Coalesce naturally behind the bounded worker queue: another request may
	// have filled this exact cover while this one waited for a slot.
	if m.covers != nil {
		if cached, ok := m.covers.read(lib.ID, cacheID); ok {
			sendCover(cached)
			return
		}
	}
	var cover []byte
	var err error
	if providerArtwork != "" && m.metadataArtwork != nil {
		var cachedPath string
		cachedPath, _, err = m.metadataArtwork.Fetch(r.Context(), providerArtwork)
		if err == nil {
			cover, err = os.ReadFile(cachedPath)
		}
		if err != nil {
			// Keep the catalogue usable during a transient provider/cache failure.
			providerArtwork = ""
			err = nil
		}
	} else if providerArtwork != "" {
		providerArtwork = ""
	}
	if providerArtwork == "" && err == nil {
		if provider, ranged := catalog.CoverCapability(lib.Store); ranged {
			cover, err = provider.Cover(r.Context(), book)
		} else {
			cover, err = archiveCover(r.Context(), lib.Store, book)
			if err != nil {
				// The file has no cover of its own: use an image lying beside it, if any.
				if sidecar, found := catalog.SidecarCoverCapability(lib.Store); found {
					if image, sidecarErr := sidecar.SidecarCover(r.Context(), book); sidecarErr == nil {
						cover, err = image, nil
					}
				}
			}
		}
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	media := "image/jpeg"
	if m.covers != nil {
		if optimized, optimizedOK := m.covers.write(lib.ID, cacheID, cover); optimizedOK {
			sendCover(optimized)
			return
		} else {
			media = http.DetectContentType(cover)
		}
	} else {
		media = http.DetectContentType(cover)
	}
	w.Header().Set("Content-Type", media)
	w.Header().Set("Cache-Control", "private, max-age=604800, immutable")
	_, _ = w.Write(cover)
}

// archiveCover extracts the cover from a publication archive. Seekable local
// files are read by ZIP index; other sources fall back to a bounded download.
func archiveCover(ctx context.Context, backend catalog.Backend, book catalog.Book) ([]byte, error) {
	opened, err := (acquisition.Service{}).Open(ctx, backend, book, "")
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	if opened.Reader != nil {
		if source, ok := opened.Reader.(io.ReaderAt); ok {
			if size, err := opened.Reader.Seek(0, io.SeekEnd); err == nil {
				if cover, _, found := artwork.FromArchiveReader(source, size); found {
					return cover, nil
				}
				return nil, fmt.Errorf("publication does not contain a cover")
			}
		}
	}
	data, err := (acquisition.Service{}).ReadAll(ctx, backend, book, 96<<20)
	if err != nil {
		return nil, err
	}
	cover, _, found := artwork.FromArchive(data)
	if !found {
		return nil, fmt.Errorf("publication does not contain a cover")
	}
	return cover, nil
}

// coverVariantWidth accepts only a few widths so the cache stays small and a
// client cannot ask for arbitrary sizes. Zero means the stored full cover.
func coverVariantWidth(value string) int {
	switch value {
	case "160":
		return 160
	case "240":
		return 240
	case "360":
		return 360
	}
	return 0
}
