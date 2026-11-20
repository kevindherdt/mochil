package video

import (
	"path/filepath"
	"testing"
)

func TestMetadataStorePersistsBoundedTMDBOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	store, err := OpenMetadataStore(path)
	if err != nil {
		t.Fatal(err)
	}
	want := MetadataOverride{Title: "Arrival", Year: 2016, Overview: "A linguist investigates.", ArtworkURL: "https://image.tmdb.org/t/p/w342/arrival.jpg", Provider: "TMDB", ProviderID: "329865"}
	if err := store.Set("movies", "one", want); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenMetadataStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.Get("movies", "one")
	if !ok || got != want {
		t.Fatalf("override=%#v exists=%v", got, ok)
	}
	show := MetadataOverride{Title: "Frieren: Beyond Journey's End", Year: 2023, Provider: "TMDB", ProviderID: "209867"}
	if err := reopened.SetShow("anime", "Sousou no Frieren", show); err != nil {
		t.Fatal(err)
	}
	reopened, err = OpenMetadataStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := reopened.GetShow("anime", "SOUSOU NO FRIEREN"); !ok || got != show {
		t.Fatalf("show override=%#v exists=%v", got, ok)
	}
	if err := reopened.DeleteShow("anime", "Sousou no Frieren"); err != nil {
		t.Fatal(err)
	}
	if err := reopened.Delete("movies", "one"); err != nil {
		t.Fatal(err)
	}
	reopened, err = OpenMetadataStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.GetShow("anime", "Sousou no Frieren"); ok {
		t.Fatal("show override survived deletion")
	}
	if _, ok := reopened.Get("movies", "one"); ok {
		t.Fatal("movie override survived deletion")
	}
}

func TestMetadataStoreRejectsUntrustedArtworkAndProvenance(t *testing.T) {
	store, err := OpenMetadataStore("")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []MetadataOverride{
		{Title: "Film", Provider: "TMDB", ProviderID: "1", ArtworkURL: "http://127.0.0.1/private.jpg"},
		{Title: "Film", Provider: "Other", ProviderID: "1"},
		{Title: "Film", Provider: "TMDB", ProviderID: "not-a-number"},
	} {
		if err := store.Set("movies", "one", value); err == nil {
			t.Fatalf("accepted unsafe override %#v", value)
		}
	}
}
