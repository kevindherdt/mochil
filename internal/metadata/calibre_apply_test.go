package metadata

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeCalibreRunner struct {
	outputs [][]byte
	errors  []error
	calls   [][]string
}

func (f *fakeCalibreRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	index := len(f.calls) - 1
	var output []byte
	if index < len(f.outputs) {
		output = f.outputs[index]
	}
	if index < len(f.errors) {
		return output, f.errors[index]
	}
	return output, nil
}

func TestResolveCalibreIDsUsesExactFormatPath(t *testing.T) {
	library := t.TempDir()
	book := filepath.Join(library, "Asato, Asato", "86 (42)", "86 - Asato.epub")
	runner := &fakeCalibreRunner{outputs: [][]byte{[]byte(`[{"id":42,"formats":["` + filepath.ToSlash(book) + `"]},{"id":7,"formats":"Elsewhere/book.epub"}]`)}}
	resolved, err := ResolveCalibreIDs(context.Background(), runner, library, []CalibreTarget{{BookPath: book}})
	if err != nil {
		t.Fatal(err)
	}
	if resolved[book] != "42" {
		t.Fatalf("resolved ID = %q", resolved[book])
	}
	want := []string{"--with-library", library, "list", "--fields", "id,formats", "--for-machine"}
	if !reflect.DeepEqual(runner.calls[0], want) {
		t.Fatalf("call = %#v, want %#v", runner.calls[0], want)
	}
}

func TestResolveCalibreIDsRefusesMissingBook(t *testing.T) {
	library := t.TempDir()
	runner := &fakeCalibreRunner{outputs: [][]byte{[]byte(`[]`)}}
	_, err := ResolveCalibreIDs(context.Background(), runner, library, []CalibreTarget{{BookPath: filepath.Join(library, "missing.epub")}})
	if err == nil || !strings.Contains(err.Error(), "could not identify") {
		t.Fatalf("expected safe resolution failure, got %v", err)
	}
}

func TestApplyCalibreVerifiesRequestedFields(t *testing.T) {
	opf := []byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Correct title</dc:title><dc:creator>Jane Doe</dc:creator><meta name="calibre:series_index" content="2.0"/></metadata></package>`)
	runner := &fakeCalibreRunner{outputs: [][]byte{nil, opf}}
	changes := map[string]string{"title": "Correct title", "authors": "Jane Doe", "series_index": "2"}
	if err := ApplyCalibre(context.Background(), runner, "/library", "42", changes); err != nil {
		t.Fatal(err)
	}
	wantSet := []string{"--with-library", "/library", "set_metadata", "42", "--field", "title:Correct title", "--field", "authors:Jane Doe", "--field", "series_index:2"}
	if !reflect.DeepEqual(runner.calls[0], wantSet) {
		t.Fatalf("set call = %#v, want %#v", runner.calls[0], wantSet)
	}
}

func TestApplyCalibreRejectsFailedVerification(t *testing.T) {
	opf := []byte(`<?xml version="1.0"?><package><metadata><title>Wrong title</title></metadata></package>`)
	runner := &fakeCalibreRunner{outputs: [][]byte{nil, opf}}
	if err := ApplyCalibre(context.Background(), runner, "/library", "42", map[string]string{"title": "Correct title"}); err == nil {
		t.Fatal("expected verification error")
	}
}

func TestBackupCalibreIsExclusiveAndRestorable(t *testing.T) {
	opf := []byte(`<?xml version="1.0"?><package><metadata><title>Original</title></metadata></package>`)
	runner := &fakeCalibreRunner{outputs: [][]byte{opf}}
	backup := filepath.Join(t.TempDir(), "job", "42.opf")
	if err := BackupCalibre(context.Background(), runner, "/library", "42", backup); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(backup)
	if err != nil || !reflect.DeepEqual(data, opf) {
		t.Fatalf("backup data = %q, err = %v", data, err)
	}
	if err := BackupCalibre(context.Background(), runner, "/library", "42", backup); err != nil {
		t.Fatalf("existing durable backup should be reusable: %v", err)
	}
	restore := &fakeCalibreRunner{}
	if err := RestoreCalibre(context.Background(), restore, "/library", "42", backup); err != nil {
		t.Fatal(err)
	}
	want := []string{"--with-library", "/library", "set_metadata", "42", backup}
	if !reflect.DeepEqual(restore.calls[0], want) {
		t.Fatalf("restore call = %#v, want %#v", restore.calls[0], want)
	}
}

func TestCalibreCommandPropagatesBoundedError(t *testing.T) {
	runner := &fakeCalibreRunner{errors: []error{errors.New("boom")}}
	if _, err := ResolveCalibreIDs(context.Background(), runner, "/library", []CalibreTarget{{BookPath: "/library/book.epub"}}); err == nil {
		t.Fatal("expected runner error")
	}
}

func TestApplyCalibreBatchIsReversible(t *testing.T) {
	library := t.TempDir()
	book := filepath.Join(library, "Writer", "Book.epub")
	backup := filepath.Join(t.TempDir(), "job", "book.opf")
	original := []byte(`<?xml version="1.0"?><package><metadata><title>Old</title></metadata></package>`)
	updated := []byte(`<?xml version="1.0"?><package><metadata><title>New</title></metadata></package>`)
	runner := &fakeCalibreRunner{outputs: [][]byte{
		[]byte(`[{"id":42,"formats":["` + filepath.ToSlash(book) + `"]}]`),
		original,
		nil,
		updated,
		nil,
		nil,
		nil,
	}}
	rollback, err := ApplyCalibreBatch(context.Background(), runner, library, []CalibreTarget{{BookPath: book, Backup: backup, Changes: map[string]string{"title": "New"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := rollback(); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 7 || runner.calls[5][2] != "set_metadata" || runner.calls[6][2] != "backup_metadata" {
		t.Fatalf("transaction calls = %#v", runner.calls)
	}
}
