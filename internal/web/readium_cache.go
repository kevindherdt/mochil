package web

import (
	"container/list"
	"fmt"
	"sync"

	"mochil/internal/catalog"
)

// derivedCache keeps the JSON the Readium routes build from an EPUB (manifest,
// positions, narration list). Building them means opening the archive and
// parsing XML, which is the slow part of opening a big book; the result only
// changes when the file does, so it is keyed on the file's size and mtime.
type derivedCache struct {
	mutex   sync.Mutex
	limit   int
	bytes   int
	maximum int
	order   *list.List
	entries map[string]*list.Element
}

type derivedEntry struct {
	key  string
	body []byte
}

func newDerivedCache(limit, maximumBytes int) *derivedCache {
	return &derivedCache{limit: limit, maximum: maximumBytes, order: list.New(), entries: map[string]*list.Element{}}
}

var readiumDerived = newDerivedCache(32, 32<<20)

func derivedKey(libraryID string, book catalog.Book, kind string) string {
	return fmt.Sprintf("%s\x00%s\x00%d\x00%d\x00%s\x00%s", libraryID, book.ID, book.Size, book.Modified.UnixNano(), book.Title, kind)
}

func (c *derivedCache) get(key string) ([]byte, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	element, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(element)
	return element.Value.(*derivedEntry).body, true
}

func (c *derivedCache) put(key string, body []byte) {
	if len(body) > c.maximum/4 {
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if element, ok := c.entries[key]; ok {
		c.bytes -= len(element.Value.(*derivedEntry).body)
		element.Value.(*derivedEntry).body = body
		c.bytes += len(body)
		c.order.MoveToFront(element)
	} else {
		c.entries[key] = c.order.PushFront(&derivedEntry{key, body})
		c.bytes += len(body)
	}
	for c.order.Len() > c.limit || c.bytes > c.maximum {
		oldest := c.order.Back()
		if oldest == nil {
			return
		}
		entry := c.order.Remove(oldest).(*derivedEntry)
		delete(c.entries, entry.key)
		c.bytes -= len(entry.body)
	}
}
