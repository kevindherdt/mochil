package web

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

// editorCollection is a collection as the editor offers it: smart ones fill themselves, so books cannot be added.
type editorCollection struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Smart bool   `json:"smart"`
	Count int    `json:"count"`
}

// memberships says which of a library's collections each of its books is in, for the signed-in person.
type memberships struct {
	names map[string][]string        // book id -> names of its collections
	books map[string]map[string]bool // collection id -> ids of its books
	list  []editorCollection
}

func (m *multiServer) editorMemberships(account string, lib Library) memberships {
	found := memberships{names: map[string][]string{}, books: map[string]map[string]bool{}}
	if m.collections == nil {
		return found
	}
	for _, c := range m.collections.list(account) {
		if c.LibraryID != lib.ID {
			continue
		}
		inside := map[string]bool{}
		for _, resolved := range m.resolveCollection(account, c) {
			if resolved.library.ID == lib.ID {
				inside[resolved.book.ID] = true
				found.names[resolved.book.ID] = append(found.names[resolved.book.ID], c.Name)
			}
		}
		found.books[c.ID] = inside
		found.list = append(found.list, editorCollection{ID: c.ID, Name: c.Name, Smart: c.Rule != nil, Count: len(inside)})
	}
	sort.SliceStable(found.list, func(i, j int) bool { return strings.ToLower(found.list[i].Name) < strings.ToLower(found.list[j].Name) })
	return found
}

// members adds books to, or takes them out of, a collection of picked books in one save. Books that belong through
// their whole series are not touched: the series is the member.
func (s *collectionStore) members(account, id, libraryID string, bookIDs []string, add bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Accounts[account] {
		c := &s.data.Accounts[account][i]
		if c.ID != id {
			continue
		}
		if c.Rule != nil {
			return 0, errSmartCollection
		}
		if c.LibraryID != "" && c.LibraryID != libraryID {
			return 0, fmt.Errorf("collection belongs to library %q", c.LibraryID)
		}
		wanted := map[string]bool{}
		for _, book := range bookIDs {
			wanted[book] = true
		}
		previous := cloneCollectionData(s.data)
		kept := make([]collectionMember, 0, len(c.Members)+len(wanted))
		changed := 0
		for _, member := range c.Members {
			if member.Type == "book" && member.LibraryID == libraryID && wanted[member.BookID] {
				if add {
					delete(wanted, member.BookID) // already in
				} else {
					changed++
					continue
				}
			}
			kept = append(kept, member)
		}
		if add {
			for _, book := range bookIDs {
				if wanted[book] {
					kept = append(kept, collectionMember{Type: "book", LibraryID: libraryID, BookID: book})
					wanted[book] = false
					changed++
				}
			}
		}
		c.Members = kept
		return changed, s.persistLocked(previous)
	}
	return 0, os.ErrNotExist
}

// editorCollections lists the library's collections (GET) and adds books to, removes them from, or creates a
// collection of the given books (POST).
func (m *multiServer) editorCollections(w http.ResponseWriter, r *http.Request) {
	if m.collections == nil {
		http.Error(w, "Collections are not available right now.", http.StatusServiceUnavailable)
		return
	}
	account := m.readingAccount(r)
	if r.Method == http.MethodGet {
		lib, ok := m.editorLibrary(w, r.URL.Query().Get("library"))
		if !ok {
			return
		}
		list := m.editorMemberships(account, lib).list
		if list == nil {
			list = []editorCollection{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"collections": list})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		Library    string   `json:"library"`
		Collection string   `json:"collection"`
		Name       string   `json:"name"`
		IDs        []string `json:"ids"`
		Remove     bool     `json:"remove"`
	}
	if err := decodeJSONRequest(w, r, 4<<20, &in, true); err != nil || len(in.IDs) == 0 {
		http.Error(w, "Pick some books first.", http.StatusBadRequest)
		return
	}
	lib, ok := m.editorLibrary(w, in.Library)
	if !ok {
		return
	}
	// Only books that exist in this library may become members: a stale or
	// cross-library id would otherwise be stored as a member that resolves to
	// nothing and still counts as a change.
	ids := make([]string, 0, len(in.IDs))
	for _, bookID := range in.IDs {
		if bookID == "" {
			continue
		}
		if _, exists := lib.Store.Book(bookID); !exists {
			http.Error(w, "One of those books is no longer in this library.", http.StatusBadRequest)
			return
		}
		ids = append(ids, bookID)
	}
	if len(ids) == 0 {
		http.Error(w, "Pick some books first.", http.StatusBadRequest)
		return
	}
	in.IDs = ids
	id := in.Collection
	if id == "" {
		name := strings.TrimSpace(in.Name)
		if name == "" || len([]rune(name)) > 80 || in.Remove {
			http.Error(w, "Give the new collection a name.", http.StatusBadRequest)
			return
		}
		// One write: creating the collection and filling it in a second save left an empty twin behind when that
		// second save failed, while the caller was told the save had failed.
		created, added, err := m.collections.createWithMembers(account, name, lib.ID, in.IDs)
		if err != nil {
			http.Error(w, "The collection could not be saved.", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"changed": added, "collection": created.ID})
		return
	}
	changed, err := m.collections.members(account, id, lib.ID, in.IDs, !in.Remove)
	switch {
	case errors.Is(err, errSmartCollection):
		http.Error(w, "A smart collection fills itself from its search, so books cannot be added or removed by hand.", http.StatusConflict)
	case errors.Is(err, os.ErrNotExist):
		http.Error(w, "That collection no longer exists.", http.StatusNotFound)
	case err != nil:
		http.Error(w, "The collection could not be saved.", http.StatusInternalServerError)
	default:
		writeJSON(w, http.StatusOK, map[string]any{"changed": changed, "collection": id})
	}
}
