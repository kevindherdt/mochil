package wiki

import (
	"testing"
	"time"
)

func TestPageQueriesDoNotExposeDeletedTimestampStorage(t *testing.T) {
	deletedAt := time.Unix(100, 0).UTC()
	store := &Store{data: storeData{Version: storeVersion, Accounts: map[string]*accountData{
		"reader": {Pages: map[string]Page{"page": {ID: "page", Title: "Page", Revision: 1, DeletedAt: &deletedAt}}},
	}}}
	page, ok := store.Get("reader", "page")
	if !ok {
		t.Fatal("stored page was not returned")
	}
	list := store.List("reader", true)
	*page.DeletedAt = page.DeletedAt.AddDate(1, 0, 0)
	*list[0].DeletedAt = time.Time{}
	stored := store.data.Accounts["reader"].Pages["page"]
	if !stored.DeletedAt.Equal(deletedAt) {
		t.Fatalf("query result mutated stored deletion timestamp: %v", stored.DeletedAt)
	}
}
