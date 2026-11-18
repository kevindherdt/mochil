package metadata

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyEPUBBacksUpValidatesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "book.epub")
	createTestEPUB(t, book, `<package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Old</dc:title><dc:creator>Unknown</dc:creator></metadata><manifest/><spine/></package>`)
	backup := filepath.Join(dir, "backups", "book.epub")
	if err := ApplyEPUB(book, backup, map[string]string{"title": "New", "authors": "Writer", "series_index": "2"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	reader, err := zip.OpenReader(book)
	if err != nil {
		t.Fatal(err)
	}
	data, err := zipEntry(reader.File, "OEBPS/content.opf")
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, value := range []string{"New", "Writer", `content="2"`} {
		if !strings.Contains(text, value) {
			t.Fatalf("updated OPF missing %q: %s", value, text)
		}
	}
	if err := RestoreEPUB(book, backup); err != nil {
		t.Fatal(err)
	}
	restored, err := zip.OpenReader(book)
	if err != nil {
		t.Fatal(err)
	}
	restoredData, err := zipEntry(restored.File, "OEBPS/content.opf")
	restored.Close()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(restoredData), ">Old<") {
		t.Fatalf("backup was not restored: %s", restoredData)
	}
}

func TestApplyEPUBRefusesExistingBackup(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "book.epub")
	backup := filepath.Join(dir, "backup.epub")
	createTestEPUB(t, book, `<package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Old</dc:title></metadata></package>`)
	if err := os.WriteFile(backup, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEPUB(book, backup, map[string]string{"title": "New"}); err == nil {
		t.Fatal("expected existing backup error")
	}
}

func TestApplyEPUBBatchCanRollbackEveryAppliedPublication(t *testing.T) {
	dir := t.TempDir()
	targets := make([]EPUBTarget, 0, 2)
	for _, name := range []string{"one", "two"} {
		book := filepath.Join(dir, name+".epub")
		createTestEPUB(t, book, `<package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Old</dc:title></metadata><manifest/><spine/></package>`)
		targets = append(targets, EPUBTarget{BookID: name, Title: name, Path: book, Backup: filepath.Join(dir, "backups", name+".epub"), Changes: map[string]string{"title": "New"}})
	}
	rollback, err := ApplyEPUBBatch(targets)
	if err != nil {
		t.Fatal(err)
	}
	if err := rollback(); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		reader, err := zip.OpenReader(target.Path)
		if err != nil {
			t.Fatal(err)
		}
		data, err := zipEntry(reader.File, "OEBPS/content.opf")
		reader.Close()
		if err != nil || !strings.Contains(string(data), ">Old<") {
			t.Fatalf("%s was not restored: %s, %v", target.Title, data, err)
		}
	}
}

func createTestEPUB(t *testing.T, path, opf string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entries := map[string]string{"mimetype": "application/epub+zip", "META-INF/container.xml": `<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`, "OEBPS/content.opf": opf}
	for _, name := range []string{"mimetype", "META-INF/container.xml", "OEBPS/content.opf"} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(entries[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyEPUBKeepsTheBooksFilePermissions(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "book.epub")
	createTestEPUB(t, book, `<package xmlns:dc="http://purl.org/dc/elements/1.1/"><metadata><dc:title>Old</dc:title></metadata><manifest/><spine/></package>`)
	if err := os.Chmod(book, 0o664); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEPUB(book, filepath.Join(dir, "backups", "book.epub"), map[string]string{"title": "New"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(book)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o664 {
		t.Fatalf("the repaired book is %v, it was 0664: other programs that read the library would lose access", info.Mode().Perm())
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, ".bobobooks-*")); len(entries) != 0 {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}
