package web

import (
	"bytes"
	"context"
	"io"
	"path"
	"strings"
	"time"

	"mochil/internal/acquisition"
	"mochil/internal/catalog"
	"mochil/internal/readpos"
)

const (
	positionBridgeTimeout = 15 * time.Second
	positionBridgeMaxEPUB = 128 << 20
)

// openEPUBPositions opens a library EPUB for position conversion.
func openEPUBPositions(lib Library, book catalog.Book) (*readpos.Book, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), positionBridgeTimeout)
	opened, err := (acquisition.Service{}).Open(ctx, lib.Store, book, "")
	if err != nil {
		cancel()
		return nil, nil, err
	}
	closeAll := func() { _ = opened.Close(); cancel() }
	if opened.Reader == nil {
		closeAll()
		return nil, nil, io.ErrUnexpectedEOF
	}
	if at, ok := opened.Reader.(io.ReaderAt); ok {
		if size, err := opened.Reader.Seek(0, io.SeekEnd); err == nil {
			epub, err := readpos.Open(at, size)
			if err != nil {
				closeAll()
				return nil, nil, err
			}
			return epub, closeAll, nil
		}
	}
	data, err := io.ReadAll(io.LimitReader(opened.Reader, positionBridgeMaxEPUB))
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	epub, err := readpos.Open(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		closeAll()
		return nil, nil, err
	}
	return epub, closeAll, nil
}

func isEPUB(book catalog.Book) bool {
	return strings.EqualFold(book.Format, "epub") || strings.EqualFold(path.Ext(book.Path), ".epub")
}

// anchorForKoReader derives the text anchor of a KOReader EPUB position so the
// web reader can open the same place. It returns nil when the position cannot
// be tied to chapter text; callers then keep only the raw locator.
func (m *multiServer) anchorForKoReader(libraryID, bookID, progress string) *webAnchor {
	lib, book, ok := m.findBook(libraryID + "/" + bookID)
	if !ok || !isEPUB(book) || lib.Store == nil {
		return nil
	}
	epub, closeEPUB, err := openEPUBPositions(lib, book)
	if err != nil {
		return nil
	}
	defer closeEPUB()
	anchor, quality, err := epub.AnchorFromXPointer(progress)
	if err != nil || quality < readpos.QualityApproximate || anchor.Exact == "" {
		return nil
	}
	return &webAnchor{Version: 2, Prefix: anchor.Prefix, Exact: anchor.Exact, Suffix: anchor.Suffix, SpineIndex: anchor.SpineIndex, Progress: anchor.Progress}
}

// koReaderProgressForWeb returns the newest web-originated position as KOReader
// progress, or false when KOReader's stored progress is already current or the
// web position cannot be converted exactly.
func (m *multiServer) koReaderProgressForWeb(username, document string, stored koSyncProgress, hasStored bool) (koSyncProgress, bool) {
	if m.reading == nil {
		return koSyncProgress{}, false
	}
	account, ok := m.koSyncAccount(username)
	if !ok {
		return koSyncProgress{}, false
	}
	libraryID, bookID, ok := m.koSyncBook(account, document)
	if !ok {
		return koSyncProgress{}, false
	}
	position, ok := m.reading.Get(account, libraryID, bookID)
	if !ok || position.WebAnchor == nil || position.WebAnchor.Exact == "" {
		return koSyncProgress{}, false
	}
	// KoSync timestamps have one-second precision.
	if hasStored && position.UpdatedAt <= stored.Timestamp*1000+999 {
		return koSyncProgress{}, false
	}
	lib, book, ok := m.findBook(libraryID + "/" + bookID)
	if !ok || !isEPUB(book) || lib.Store == nil {
		return koSyncProgress{}, false
	}
	epub, closeEPUB, err := openEPUBPositions(lib, book)
	if err != nil {
		return koSyncProgress{}, false
	}
	defer closeEPUB()
	anchor := readpos.Anchor{SpineIndex: position.WebAnchor.SpineIndex, Prefix: position.WebAnchor.Prefix, Exact: position.WebAnchor.Exact, Suffix: position.WebAnchor.Suffix, Progress: position.WebAnchor.Progress}
	if anchor.Exact == "" {
		anchor.Exact = position.WebAnchor.Text
	}
	xpointer, quality, err := epub.XPointerFromAnchor(anchor)
	if err != nil || quality != readpos.QualityText {
		return koSyncProgress{}, false
	}
	device := position.DeviceName
	if device == "" {
		device = "Mochil Web"
	}
	return koSyncProgress{
		Document: document, Progress: xpointer, Percentage: position.Fraction,
		Device: device, DeviceID: position.DeviceID, Timestamp: position.UpdatedAt / 1000,
	}, true
}
