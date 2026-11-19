package web

import "mochil/internal/catalog"

type metadataApplyEffects struct {
	Scanned int
	Warning string
}

// finishMetadataSourceApply owns the runtime effects that follow a committed
// source transaction. Keeping this out of the HTTP handlers ensures every
// writable adapter invalidates affected covers before publishing its rescan.
func (m *multiServer) finishMetadataSourceApply(library Library, books map[string]catalog.Book) metadataApplyEffects {
	if m.covers != nil {
		for _, book := range books {
			m.covers.invalidate(library.ID, book.ID)
		}
	}
	stats, err := library.Store.Scan()
	effects := metadataApplyEffects{Scanned: stats.Books}
	if err != nil {
		effects.Warning = err.Error()
	}
	return effects
}
