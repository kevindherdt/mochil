package ingest

import (
	"archive/zip"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mochil/internal/opf"
)

const (
	dublinCoreNamespace = "http://purl.org/dc/elements/1.1/"
	opfNamespace        = "http://www.idpf.org/2007/opf"
)

var (
	metadataElement = regexp.MustCompile(`(?is)(<(?:[\w-]+:)?metadata\b)([^>]*)(>)(.*?)(</(?:[\w-]+:)?metadata\s*>)`)
	metaElement     = regexp.MustCompile(`(?is)<meta\b([^>]*?)(?:/>|>(.*?)</meta\s*>)`)
	idAttribute     = regexp.MustCompile(`(?is)\bid\s*=\s*["']([^"']+)["']`)
)

// EmbedMetadata writes a book's fields into the package document of an EPUB that is already inside the managed
// library. It is meant for the copy Mochil made: the original in the drop folder stays as it was, so it is the
// backup. Only fields that have a value are written; everything else in the package, and every other file inside the
// EPUB, is carried over unchanged. The result is read back before the copy is replaced.
func (e *Executor) EmbedMetadata(targetPath string, meta opf.Metadata) error {
	if !safeJournalPath(targetPath) || strings.HasPrefix(targetPath, ".ingest/") {
		return errors.New("the path is not a managed library path")
	}
	if !strings.EqualFold(filepath.Ext(targetPath), ".epub") {
		return errors.New("only EPUB files can have metadata embedded")
	}
	if strings.TrimSpace(meta.Title) == "" {
		return errors.New("a book needs a title to embed")
	}
	resolved, err := filepath.EvalSymlinks(e.target(targetPath))
	if err != nil {
		return errors.New("the book is not in the managed library")
	}
	if !withinRoot(e.targetRoot, resolved) {
		return errors.New("the book resolves outside the managed library")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return embedEPUB(resolved, meta)
}

func embedEPUB(path string, meta opf.Metadata) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("the book must be a regular file")
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("the EPUB cannot be read: %w", err)
	}
	defer reader.Close()
	opfPath, err := rootfile(reader)
	if err != nil {
		return err
	}
	raw, err := readZipEntry(reader, opfPath)
	if err != nil {
		return err
	}
	updated, err := embedInPackage(string(raw), meta)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".mochil-embed-*.epub")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := writeEPUBWithPackage(reader, temporary, opfPath, updated); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := verifyEmbedded(temporaryPath, meta); err != nil {
		return fmt.Errorf("the rewritten EPUB did not check out, so the copy was left as it was: %w", err)
	}
	return os.Rename(temporaryPath, path)
}

// writeEPUBWithPackage copies every entry, in order and with its own header (so the stored "mimetype" stays first and
// uncompressed), replacing only the package document.
//
// Unchanged entries are copied in their compressed form. Decompressing and
// recompressing them was both slow and an amplifier: an 8 KB archive could carry
// an entry that expands to gigabytes, and the rewrite inflated and deflated it
// just to copy it back. Rows go through CreateRaw, so the bytes never leave the
// archive's own compression.
func writeEPUBWithPackage(source *zip.ReadCloser, output io.Writer, opfPath, packageText string) error {
	writer := zip.NewWriter(output)
	for _, entry := range source.File {
		header := entry.FileHeader
		if path.Clean(entry.Name) == path.Clean(opfPath) {
			destination, err := writer.CreateHeader(&header)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(destination, packageText); err != nil {
				return err
			}
			continue
		}
		if err := copyEntryUnchanged(writer, entry); err != nil {
			return err
		}
	}
	return writer.Close()
}

// copyEntryUnchanged writes one entry back exactly as it was stored.
func copyEntryUnchanged(writer *zip.Writer, entry *zip.File) error {
	header := entry.FileHeader
	if entry.FileInfo().IsDir() {
		// A directory has no compressed stream to copy.
		_, err := writer.CreateHeader(&header)
		return err
	}
	raw, err := entry.OpenRaw()
	if err != nil {
		return err
	}
	destination, err := writer.CreateRaw(&header)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, raw)
	return copyErr
}

func verifyEmbedded(path string, meta opf.Metadata) error {
	got, err := ReadEmbedded(path)
	if err != nil {
		return err
	}
	if got.Title != strings.TrimSpace(meta.Title) {
		return errors.New("the title did not survive")
	}
	if len(meta.Authors) > 0 && strings.Join(got.Authors, "|") != strings.Join(trimmed(meta.Authors), "|") {
		return errors.New("the authors did not survive")
	}
	if strings.TrimSpace(meta.Series) != "" {
		if got.Series != strings.TrimSpace(meta.Series) {
			return errors.New("the series did not survive")
		}
		if meta.SeriesIndex != nil && (got.SeriesIndex == nil || *got.SeriesIndex != *meta.SeriesIndex) {
			return errors.New("the series number did not survive")
		}
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) == 0 || reader.File[0].Name != "mimetype" {
		return errors.New("the mimetype entry is no longer first")
	}
	return nil
}

func trimmed(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

// embedInPackage returns the package document with the book's fields written into its metadata block.
func embedInPackage(document string, meta opf.Metadata) (string, error) {
	if !strings.Contains(document, dublinCoreNamespace) {
		return "", errors.New("the package does not declare the Dublin Core namespace, so it was not changed")
	}
	match := metadataElement.FindStringSubmatchIndex(document)
	if match == nil {
		return "", errors.New("the package has no metadata block")
	}
	open, attributes, close, inner, end := document[match[2]:match[3]], document[match[4]:match[5]], document[match[6]:match[7]], document[match[8]:match[9]], document[match[10]:match[11]]
	if !strings.Contains(document, `xmlns:opf=`) && !strings.Contains(attributes, `xmlns:opf=`) {
		attributes += ` xmlns:opf="` + opfNamespace + `"`
	}
	var added []string
	add := func(line string) { added = append(added, "    "+line) }
	text := func(value string) string { return html.EscapeString(strings.TrimSpace(value)) }
	removeMeta := func(drop func(attributes string) bool) []string {
		var ids []string
		inner = metaElement.ReplaceAllStringFunc(inner, func(found string) string {
			parts := metaElement.FindStringSubmatch(found)
			if !drop(parts[1]) {
				return found
			}
			if id := idAttribute.FindStringSubmatch(parts[1]); id != nil {
				ids = append(ids, id[1])
			}
			return ""
		})
		return ids
	}
	// removeDC deletes every dc:<name> element, and any EPUB 3 refinement of one that had an id.
	removeDC := func(name string) {
		pattern := regexp.MustCompile(`(?is)\s*<dc:` + name + `\b([^>]*?)(?:/>|>.*?</dc:` + name + `\s*>)`)
		var ids []string
		for _, found := range pattern.FindAllStringSubmatch(inner, -1) {
			if id := idAttribute.FindStringSubmatch(found[1]); id != nil {
				ids = append(ids, id[1])
			}
		}
		inner = pattern.ReplaceAllString(inner, "")
		dropRefinements(&inner, ids)
	}
	dropByName := func(name string) {
		removeMeta(func(a string) bool { return hasAttribute(a, "name", name) })
	}

	add("<dc:title>" + text(meta.Title) + "</dc:title>")
	removeDC("title")
	if authors := trimmed(meta.Authors); len(authors) > 0 {
		removeDC("creator")
		for _, author := range authors {
			add(`<dc:creator opf:role="aut">` + text(author) + `</dc:creator>`)
		}
	}
	if series := strings.TrimSpace(meta.Series); series != "" {
		dropByName("calibre:series")
		dropByName("calibre:series_index")
		ids := removeMeta(func(a string) bool { return hasAttribute(a, "property", "belongs-to-collection") })
		dropRefinements(&inner, ids)
		add(`<meta name="calibre:series" content="` + html.EscapeString(series) + `"/>`)
		if meta.SeriesIndex != nil {
			add(`<meta name="calibre:series_index" content="` + strconv.FormatFloat(*meta.SeriesIndex, 'f', -1, 64) + `"/>`)
		}
	}
	if tags := trimmed(meta.Tags); len(tags) > 0 {
		removeDC("subject")
		for _, tag := range tags {
			add("<dc:subject>" + text(tag) + "</dc:subject>")
		}
	}
	if strings.TrimSpace(meta.Publisher) != "" {
		removeDC("publisher")
		add("<dc:publisher>" + text(meta.Publisher) + "</dc:publisher>")
	}
	if strings.TrimSpace(meta.Published) != "" {
		removeDC("date")
		add("<dc:date>" + text(meta.Published) + "</dc:date>")
	}
	if strings.TrimSpace(meta.Language) != "" {
		removeDC("language")
		add("<dc:language>" + text(meta.Language) + "</dc:language>")
	}
	if strings.TrimSpace(meta.Description) != "" {
		removeDC("description")
		add("<dc:description>" + text(meta.Description) + "</dc:description>")
	}
	if meta.Rating > 0 {
		dropByName("calibre:rating")
		add(`<meta name="calibre:rating" content="` + strconv.Itoa(meta.Rating) + `"/>`)
	}
	if strings.TrimSpace(meta.TitleSort) != "" {
		dropByName("calibre:title_sort")
		add(`<meta name="calibre:title_sort" content="` + html.EscapeString(strings.TrimSpace(meta.TitleSort)) + `"/>`)
	}
	// Identifiers the book already has are kept, whatever their scheme: the package's unique-identifier points at one.
	for _, scheme := range sortedKeys(meta.Identifiers) {
		value := strings.TrimSpace(meta.Identifiers[scheme])
		if value == "" || strings.Contains(strings.ToLower(inner), strings.ToLower(html.EscapeString(value))) {
			continue
		}
		add(`<dc:identifier opf:scheme="` + html.EscapeString(strings.ToUpper(scheme)) + `">` + text(value) + `</dc:identifier>`)
	}
	inner = strings.TrimRight(inner, " \t\r\n") + "\n" + strings.Join(added, "\n") + "\n  "
	return document[:match[0]] + open + attributes + close + inner + end + document[match[1]:], nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func hasAttribute(attributes, name, value string) bool {
	pattern := regexp.MustCompile(`(?is)\b` + regexp.QuoteMeta(name) + `\s*=\s*["']` + regexp.QuoteMeta(value) + `["']`)
	return pattern.MatchString(attributes)
}

// dropRefinements removes EPUB 3 <meta refines="#id"> records that described an element that has been removed.
func dropRefinements(inner *string, ids []string) {
	for _, id := range ids {
		pattern := regexp.MustCompile(`(?is)\s*<meta\b[^>]*\brefines\s*=\s*["']#` + regexp.QuoteMeta(id) + `["'][^>]*?(?:/>|>.*?</meta\s*>)`)
		*inner = pattern.ReplaceAllString(*inner, "")
	}
}
