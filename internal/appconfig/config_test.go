package appconfig

import (
	"strings"
	"testing"
	"time"
)

func TestValidateRejectsDuplicateLibraryIDs(t *testing.T) {
	cfg := Config{
		ScanEvery: "1m",
		Libraries: []Library{
			{ID: "books", Kind: "books", Source: Source{Type: "filesystem", Path: "/one"}, Inference: Inference{FolderLayout: "auto"}},
			{ID: "books", Kind: "books", Source: Source{Type: "filesystem", Path: "/two"}, Inference: Inference{FolderLayout: "auto"}},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate ID error")
	}
}

func TestValidateRejectsRemovedSeafileSource(t *testing.T) {
	cfg := Config{ScanEvery: "1m", Libraries: []Library{{
		ID: "books", Kind: "books",
		Source:    Source{Type: "seafile"},
		Inference: Inference{FolderLayout: "series"},
	}}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "filesystem or calibre") {
		t.Fatalf("expected Seafile source to be rejected, got %v", err)
	}
}

func TestValidateRejectsCalibreDocuments(t *testing.T) {
	cfg := Config{ScanEvery: "1m", Libraries: []Library{{ID: "documents", Kind: "documents", Source: Source{Type: "calibre", Path: "/library"}, Inference: Inference{FolderLayout: "auto"}}}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "require a filesystem source") {
		t.Fatalf("expected document source validation error, got %v", err)
	}
}

func TestDecodeAppliesBoundedCoverCacheDefaults(t *testing.T) {
	cfg, err := Decode(strings.NewReader(`{"libraries":[{"id":"books","source":{"type":"filesystem","path":"/books"}}]}`), "/config")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CoverCache.MaximumBytes != 512<<20 || cfg.CoverCache.MaximumWidth != 600 || cfg.CoverCache.MaximumHeight != 900 || cfg.CoverCache.JPEGQuality != 82 {
		t.Fatalf("unexpected cover cache defaults: %+v", cfg.CoverCache)
	}
	if got := strings.Join(cfg.Libraries[0].PreferredFormats, ","); got != "epub,pdf,mobi,azw3" {
		t.Fatalf("preferred format defaults = %q", got)
	}
	if cfg.Playback.WiFiQuality != "maximum" || cfg.Playback.MobileQuality != "high" {
		t.Fatalf("unexpected playback quality defaults: %+v", cfg.Playback)
	}
}

func TestValidateRejectsUnknownPlaybackQuality(t *testing.T) {
	cfg := Config{ScanEvery: "1m", Playback: Playback{WiFiQuality: "lossless", MobileQuality: "high"}, Libraries: []Library{{ID: "music", Kind: "music", Source: Source{Type: "filesystem", Path: "/music"}, Inference: Inference{FolderLayout: "auto"}}}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "playback quality") {
		t.Fatalf("expected playback quality validation error, got %v", err)
	}
}

func TestDecodeAcceptsAndDiscardsRetiredExternalSources(t *testing.T) {
	cfg, err := Decode(strings.NewReader(`{"external_sources":{"download_directory":"source-downloads"},"libraries":[{"id":"books","source":{"type":"filesystem","path":"/books"}}]}`), "/config")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.RetiredExternalSources) != 0 {
		t.Fatalf("legacy source configuration survived decoding: %q", cfg.RetiredExternalSources)
	}
}

func TestDecodeAcceptsAndIgnoresRetiredLoginBackground(t *testing.T) {
	for _, value := range []string{"glow", "spines", "purple"} {
		cfg, err := Decode(strings.NewReader(`{"appearance":{"login_background":"`+value+`"},"libraries":[{"id":"books","source":{"type":"filesystem","path":"/books"}}]}`), "/config")
		if err != nil {
			t.Fatalf("login_background %q: %v", value, err)
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("login_background %q failed validation: %v", value, err)
		}
		if cfg.Appearance.LoginBackground != nil {
			t.Fatalf("retired login_background %q survived decoding: %v", value, cfg.Appearance.LoginBackground)
		}
	}
}

func TestDecodeResolvesManagedIngestRootsAndDefaultLimit(t *testing.T) {
	cfg, err := Decode(strings.NewReader(`{"managed_ingest":{"incoming_directory":"incoming","managed_directory":"managed"},"libraries":[{"id":"books","source":{"type":"filesystem","path":"/books"}}]}`), "/config")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ManagedIngest.IncomingDirectory != "/config/incoming" || cfg.ManagedIngest.ManagedDirectory != "/config/managed" || cfg.ManagedIngest.MaximumFileBytes != 8<<30 {
		t.Fatalf("unexpected managed ingest configuration: %+v", cfg.ManagedIngest)
	}
}

// Picking a drop folder is the whole setup step a reader should need: the managed
// folder only ever holds Mochil's own verified copies, so it is derived rather
// than asked for. It must land under cache_directory (the precious, backed-up
// zone despite the name — see bootstrap.json) and never inside the drop folder
// itself, or a copy already made would look like a new file to copy again.
func TestDecodeDerivesTheManagedFolderWhenOnlyTheDropFolderIsSet(t *testing.T) {
	cfg, err := Decode(strings.NewReader(`{"cache_directory":"/data/catalogs","managed_ingest":{"incoming_directory":"/media/incoming"},"libraries":[{"id":"books","source":{"type":"filesystem","path":"/books"}}]}`), "/config")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ManagedIngest.IncomingDirectory != "/media/incoming" {
		t.Fatalf("incoming directory changed unexpectedly: %q", cfg.ManagedIngest.IncomingDirectory)
	}
	if cfg.ManagedIngest.ManagedDirectory != "/data/catalogs/managed-ingest" {
		t.Fatalf("managed directory = %q, want it derived under cache_directory", cfg.ManagedIngest.ManagedDirectory)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("derived managed_ingest should validate cleanly: %v", err)
	}
}

func TestValidateManagedIngestRequiresSeparateCompleteRoots(t *testing.T) {
	base := Config{ScanEvery: "1m", Libraries: []Library{{ID: "books", Kind: "books", Source: Source{Type: "filesystem", Path: "/books"}, Inference: Inference{FolderLayout: "auto"}}}}
	for name, managed := range map[string]ManagedIngest{
		"missing managed":  {IncomingDirectory: "/incoming"},
		"missing incoming": {ManagedDirectory: "/managed"},
		"same root":        {IncomingDirectory: "/media", ManagedDirectory: "/media", MaximumFileBytes: 1 << 30},
		"managed nested":   {IncomingDirectory: "/media", ManagedDirectory: "/media/library", MaximumFileBytes: 1 << 30},
		"incoming nested":  {IncomingDirectory: "/media/drop", ManagedDirectory: "/media", MaximumFileBytes: 1 << 30},
		"too small":        {IncomingDirectory: "/incoming", ManagedDirectory: "/managed", MaximumFileBytes: 1024},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := base
			cfg.ManagedIngest = managed
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid managed ingest configuration was accepted")
			}
		})
	}
}

func TestValidateRejectsInvalidPreferredFormats(t *testing.T) {
	base := Library{ID: "books", Kind: "books", Source: Source{Type: "filesystem", Path: "/books"}, Inference: Inference{FolderLayout: "auto"}}
	for _, formats := range [][]string{{"epub", "exe"}, {"epub", ".EPUB"}} {
		library := base
		library.PreferredFormats = formats
		cfg := Config{ScanEvery: "1m", Libraries: []Library{library}}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "preferred_formats") {
			t.Fatalf("formats %v: expected validation error, got %v", formats, err)
		}
	}
}

func TestDecodePreservesExplicitlyEmptyPreferredFormats(t *testing.T) {
	cfg, err := Decode(strings.NewReader(`{"libraries":[{"id":"books","preferred_formats":[],"source":{"type":"filesystem","path":"/books"}}]}`), "/config")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Libraries[0].PreferredFormats == nil || len(cfg.Libraries[0].PreferredFormats) != 0 {
		t.Fatalf("explicit empty preferred formats were replaced: %#v", cfg.Libraries[0].PreferredFormats)
	}
}

func TestValidateAllowsConfiguredMediaLibraries(t *testing.T) {
	for _, kind := range []string{"movies", "tv", "anime", "audiobooks", "podcasts", "music", "documents", "vault"} {
		cfg := Config{ScanEvery: "1m", Libraries: []Library{{ID: kind, Kind: kind, Source: Source{Type: "filesystem", Path: "/media/" + kind}, Inference: Inference{FolderLayout: "auto"}}}}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("%s library should be configurable: %v", kind, err)
		}
	}
}

func TestValidateAllowsFirstRunWithoutLibraries(t *testing.T) {
	cfg := Config{ScanEvery: "10m"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("empty first-run configuration should be valid: %v", err)
	}
}

func TestValidateMetadataProvidersMatchLibraryKinds(t *testing.T) {
	tests := []struct {
		name, kind, provider string
		providers            MetadataProviders
		valid                bool
	}{
		{name: "open library books", kind: "books", provider: "openlibrary", valid: true},
		{name: "hardcover manga", kind: "manga", provider: "hardcover", providers: MetadataProviders{HardcoverToken: "secret"}, valid: true},
		{name: "tmdb television", kind: "tv", provider: "tmdb", providers: MetadataProviders{TMDBToken: "secret"}, valid: true},
		{name: "tmdb books", kind: "books", provider: "tmdb", providers: MetadataProviders{TMDBToken: "secret"}},
		{name: "open library movies", kind: "movies", provider: "openlibrary"},
		{name: "hardcover without token", kind: "books", provider: "hardcover"},
		{name: "tmdb without token", kind: "movies", provider: "tmdb"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := Config{ScanEvery: "1m", MetadataProviders: test.providers, Libraries: []Library{{
				ID: "media", Kind: test.kind, MetadataProvider: test.provider,
				Source: Source{Type: "filesystem", Path: "/media"}, Inference: Inference{FolderLayout: "auto"},
			}}}
			err := cfg.Validate()
			if test.valid && err != nil {
				t.Fatalf("expected valid provider: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("expected provider validation error")
			}
		})
	}
}

func TestValidateAllowsMangaLibraries(t *testing.T) {
	cfg := Config{ScanEvery: "1m", Libraries: []Library{{ID: "manga", Kind: "manga", Source: Source{Type: "filesystem", Path: "/media/manga"}, Inference: Inference{FolderLayout: "series"}}}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("manga library should be configurable: %v", err)
	}
}

func TestValidateAllowsReadOnlyCalibreLibrary(t *testing.T) {
	cfg := Config{ScanEvery: "1m", Libraries: []Library{{ID: "calibre", Kind: "books", Source: Source{Type: "calibre", Path: "/media/Calibre Library"}, Inference: Inference{FolderLayout: "metadata-only"}}}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Calibre library should be configurable: %v", err)
	}
}

func TestValidateAllowsWritableCalibreLibrary(t *testing.T) {
	cfg := Config{ScanEvery: "1m", Libraries: []Library{{ID: "calibre", Kind: "books", MetadataWrites: true, Source: Source{Type: "calibre", Path: "/media/Calibre Library"}, Inference: Inference{FolderLayout: "metadata-only"}}}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("writable Calibre library should be configurable: %v", err)
	}
}

func TestMetadataWritesRequireFilesystemLibrary(t *testing.T) {
	cfg := Config{ScanEvery: "1m", Libraries: []Library{{ID: "docs", Kind: "movies", MetadataWrites: true, Source: Source{Type: "filesystem", Path: "/media"}, Inference: Inference{FolderLayout: "auto"}}}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "publication libraries") {
		t.Fatalf("expected metadata write validation error, got %v", err)
	}
}

func TestModulesFollowLibrariesUnlessSetExplicitly(t *testing.T) {
	books := Library{ID: "books", Kind: "books"}
	movies := Library{ID: "films", Kind: "movies"}
	cfg := Config{Libraries: []Library{books}}
	state := cfg.EnabledModules()
	if state["video"] || state["audio"] || state["documents"] {
		t.Fatalf("an install with only books must run no optional module: %v", state)
	}
	cfg.Libraries = append(cfg.Libraries, movies)
	if !cfg.ModuleEnabled("video") || cfg.ModuleEnabled("audio") {
		t.Fatalf("a movie library turns on video only: %v", cfg.EnabledModules())
	}
	cfg.Modules = map[string]bool{"video": false, "audio": true}
	if cfg.ModuleEnabled("video") || !cfg.ModuleEnabled("audio") {
		t.Fatalf("explicit settings must win: %v", cfg.EnabledModules())
	}
	if ModuleForKind("books") != "" || ModuleForKind("anime") != "video" || ModuleForKind("vault") != "documents" || ModuleForKind("podcasts") != "audio" {
		t.Fatal("unexpected kind mapping")
	}
}

func TestUnknownModulesAreRejected(t *testing.T) {
	cfg := Config{Modules: map[string]bool{"teleport": true}}
	if err := cfg.validateModules(); err == nil {
		t.Fatal("unknown module must be rejected")
	}
}

func TestLibraryScanIntervalOverridesTheGlobalOne(t *testing.T) {
	const global = 10 * time.Minute
	for _, test := range []struct {
		setting string
		want    time.Duration
	}{{"", global}, {"  ", global}, {"0", 0}, {"1m", time.Minute}, {"6h", 6 * time.Hour}} {
		got, err := Library{ScanEvery: test.setting}.ScanDuration(global)
		if err != nil || got != test.want {
			t.Errorf("scan_every %q = %v, %v; want %v", test.setting, got, err, test.want)
		}
	}
}

func TestValidateRejectsBadLibraryScanIntervals(t *testing.T) {
	for _, setting := range []string{"soon", "-5m", "10s"} {
		cfg := Config{ScanEvery: "1m", Libraries: []Library{{ID: "books", Kind: "books", ScanEvery: setting,
			Source: Source{Type: "filesystem", Path: "/one"}, Inference: Inference{FolderLayout: "auto"}}}}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "scan_every") {
			t.Errorf("scan_every %q was accepted: %v", setting, err)
		}
	}
}

// The online metadata provider is no longer a user choice: a library that does
// not name one gets the provider its kind and tokens allow, at decode time.
func TestDecodeDerivesLibraryMetadataProvider(t *testing.T) {
	const tokens = `"metadata_providers":{"hardcover_token":"book-secret","tmdb_token":"video-secret"}`
	tests := []struct {
		name, kind, explicit string
		providers            string
		want                 string
	}{
		{name: "books with a hardcover token", kind: "books", providers: tokens, want: "hardcover"},
		{name: "books without tokens", kind: "books", want: "openlibrary"},
		{name: "comics with a hardcover token", kind: "comics", providers: tokens, want: "hardcover"},
		{name: "manga with a hardcover token", kind: "manga", providers: tokens, want: "hardcover"},
		{name: "mixed without tokens", kind: "mixed", want: "openlibrary"},
		{name: "movies with a tmdb token", kind: "movies", providers: tokens, want: "tmdb"},
		{name: "movies without a tmdb token", kind: "movies", providers: `"metadata_providers":{"hardcover_token":"book-secret"}`, want: ""},
		{name: "tv with a tmdb token", kind: "tv", providers: tokens, want: "tmdb"},
		{name: "anime without tokens", kind: "anime", want: ""},
		{name: "vault", kind: "vault", providers: tokens, want: ""},
		{name: "documents", kind: "documents", providers: tokens, want: ""},
		{name: "audiobooks", kind: "audiobooks", providers: tokens, want: ""},
		{name: "auto picks hardcover", kind: "books", explicit: `"metadata_provider":"auto"`, providers: tokens, want: "hardcover"},
		{name: "an explicit hardcover stays", kind: "books", explicit: `"metadata_provider":"hardcover"`, providers: tokens, want: "hardcover"},
		{name: "an explicit open library outranks the token", kind: "books", explicit: `"metadata_provider":"openlibrary"`, providers: tokens, want: "openlibrary"},
		{name: "off means no provider", kind: "books", explicit: `"metadata_provider":"off"`, providers: tokens, want: ""},
		{name: "disabled means no provider", kind: "movies", explicit: `"metadata_provider":"disabled"`, providers: tokens, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			providers := test.providers
			if providers != "" {
				providers = "," + providers
			}
			explicit := ""
			if test.explicit != "" {
				explicit = "," + test.explicit
			}
			document := `{"libraries":[{"id":"library","kind":"` + test.kind + `"` + explicit + `,"source":{"type":"filesystem","path":"/media/library"}}]` + providers + `}`
			cfg, err := Decode(strings.NewReader(document), "/config")
			if err != nil {
				t.Fatalf("a derived provider must never fail validation: %v", err)
			}
			if got := cfg.Libraries[0].MetadataProvider; got != test.want {
				t.Fatalf("%s metadata_provider = %q, want %q", test.kind, got, test.want)
			}
		})
	}
}

func TestDerivedMetadataProviderNeverOutrunsItsToken(t *testing.T) {
	cfg, err := Decode(strings.NewReader(`{"libraries":[{"id":"films","kind":"movies","source":{"type":"filesystem","path":"/media/films"}}]}`), "/config")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Libraries[0].MetadataProvider != "" {
		t.Fatalf("movies without a TMDB token got %q, which the runtime cannot use", cfg.Libraries[0].MetadataProvider)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("derived configuration must always validate: %v", err)
	}
}

// A leftover explicit provider must never be able to block a save now that the
// settings page cannot edit it: the field fell out of the form, so a value that
// no longer fits the library is replaced instead of failing validation.
func TestDecodeReplacesAnExplicitProviderThatNoLongerFits(t *testing.T) {
	const tokens = `"metadata_providers":{"hardcover_token":"book-secret","tmdb_token":"video-secret"}`
	for _, test := range []struct {
		name      string
		kind      string
		explicit  string
		providers string
		want      string
	}{
		{name: "hardcover on an audiobook library", kind: "audiobooks", explicit: `"metadata_provider":"hardcover"`, providers: tokens, want: ""},
		{name: "tmdb on a books library", kind: "books", explicit: `"metadata_provider":"tmdb"`, providers: tokens, want: "hardcover"},
		{name: "hardcover with no token left", kind: "books", explicit: `"metadata_provider":"hardcover"`, providers: `"metadata_providers":{}`, want: "openlibrary"},
		{name: "tmdb with no token left", kind: "movies", explicit: `"metadata_provider":"tmdb"`, providers: `"metadata_providers":{}`, want: ""},
		{name: "hardcover that still fits", kind: "books", explicit: `"metadata_provider":"hardcover"`, providers: tokens, want: "hardcover"},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := `{"scan_every":"1m","libraries":[{"id":"media","name":"Media","kind":"` + test.kind + `",` +
				test.explicit + `,"source":{"type":"filesystem","path":"/media"},"inference":{"folder_layout":"auto"}}],` +
				test.providers + `}`
			cfg, err := Decode(strings.NewReader(document), t.TempDir())
			if err != nil {
				t.Fatalf("decode: %v (a stale provider must not block a save)", err)
			}
			if got := cfg.Libraries[0].MetadataProvider; got != test.want {
				t.Fatalf("metadata_provider = %q, want %q", got, test.want)
			}
		})
	}
}
