package metadata

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type EPUBTarget struct {
	BookID  string
	Title   string
	Path    string
	Backup  string
	Changes map[string]string
}

// ApplyEPUBBatch preflights the complete set before writing, applies targets in
// deterministic path order, and restores earlier publications in reverse order
// if any update fails. The returned rollback participates in a wider
// transaction with catalogue overrides and review-job state.
func ApplyEPUBBatch(targets []EPUBTarget) (func() error, error) {
	if len(targets) == 0 {
		return func() error { return nil }, nil
	}
	ordered := append([]EPUBTarget(nil), targets...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for _, target := range ordered {
		if _, err := os.Stat(target.Backup); err == nil {
			return nil, errors.New("a backup for this job already exists; refusing to overwrite it")
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("inspect backup for %s: %w", target.Title, err)
		}
		if err := PreflightEPUB(target.Path, target.Changes); err != nil {
			return nil, fmt.Errorf("preflight failed for %s: %w", target.Title, err)
		}
	}
	applied := make([]EPUBTarget, 0, len(ordered))
	rollback := func() error {
		var failures []string
		for index := len(applied) - 1; index >= 0; index-- {
			if err := RestoreEPUB(applied[index].Path, applied[index].Backup); err != nil {
				failures = append(failures, err.Error())
			}
		}
		if len(failures) != 0 {
			return errors.New(strings.Join(failures, "; "))
		}
		return nil
	}
	for _, target := range ordered {
		if err := ApplyEPUB(target.Path, target.Backup, target.Changes); err != nil {
			if rollbackErr := rollback(); rollbackErr != nil {
				return nil, &TransactionError{Err: fmt.Errorf("apply failed and rollback needs attention: %w", rollbackErr)}
			}
			return nil, fmt.Errorf("apply failed; earlier changes were restored: %w", err)
		}
		applied = append(applied, target)
	}
	return rollback, nil
}

const maximumPackageBytes = 8 << 20

var (
	titleElement   = regexp.MustCompile(`(?is)(<dc:title(?:\s[^>]*)?>)(.*?)(</dc:title>)`)
	creatorElement = regexp.MustCompile(`(?is)(<dc:creator(?:\s[^>]*)?>)(.*?)(</dc:creator>)`)
	// The tag is matched on its own and the content attribute is rewritten inside it,
	// so both attribute orders count. Matching name-before-content missed the
	// `<meta content="2" name="calibre:series_index"/>` order and appended a second,
	// contradicting index instead of correcting the existing one.
	seriesIndexTag     = regexp.MustCompile(`(?is)<meta\s+[^>]*name=["']calibre:series_index["'][^>]*?/?>`)
	seriesIndexContent = regexp.MustCompile(`(?is)(content=["'])(.*?)(["'])`)
)

// ApplyEPUB rewrites a single ordinary-folder EPUB using an adjacent temporary
// file. It validates the result before atomically replacing the original and
// stores a durable backup outside the library first.
func ApplyEPUB(path, backupPath string, changes map[string]string) error {
	if strings.ToLower(filepath.Ext(path)) != ".epub" {
		return errors.New("metadata repair supports EPUB files only")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("publication must be a regular non-symlink file")
	}
	if len(changes) == 0 {
		return errors.New("metadata repair has no changes")
	}
	if err := createExclusiveBackup(path, backupPath, info.Mode()); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".mochil-repair-*.epub")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	if err := temporary.Close(); err != nil {
		return err
	}
	defer os.Remove(temporaryPath)
	if err := rewriteEPUB(path, temporaryPath, changes, info.Mode()); err != nil {
		return err
	}
	if err := validateEPUB(temporaryPath); err != nil {
		return fmt.Errorf("validate rewritten EPUB: %w", err)
	}
	// The original is already backed up outside the library, so one rename replaces it atomically: the book is never
	// missing, even for a moment.
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace publication: %w", err)
	}
	return nil
}

// RestoreEPUB restores a validated backup through an adjacent temporary file.
func RestoreEPUB(path, backupPath string) error {
	if err := validateEPUB(backupPath); err != nil {
		return fmt.Errorf("validate backup: %w", err)
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".mochil-restore-*.epub")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	input, err := os.Open(backupPath)
	if err != nil {
		temporary.Close()
		return err
	}
	_, copyErr := io.Copy(temporary, input)
	input.Close()
	if copyErr != nil {
		temporary.Close()
		return copyErr
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
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}

// PreflightEPUB validates the source and proposed package changes without writing.
func PreflightEPUB(path string, changes map[string]string) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("publication must be a regular non-symlink file")
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	opfPath, err := packagePath(reader.File)
	if err != nil {
		return err
	}
	data, err := zipEntry(reader.File, opfPath)
	if err != nil {
		return err
	}
	updated, err := updatePackage(string(data), changes)
	if err != nil {
		return err
	}
	var document struct{ XMLName xml.Name }
	return xml.Unmarshal([]byte(updated), &document)
}

func rewriteEPUB(source, target string, changes map[string]string, mode os.FileMode) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()
	opfPath, err := packagePath(reader.File)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	// The temporary file was made owner-only; the book keeps the permissions it had.
	if err := output.Chmod(mode.Perm()); err != nil {
		output.Close()
		return err
	}
	writer := zip.NewWriter(output)
	found := false
	for _, entry := range reader.File {
		header := entry.FileHeader
		destination, createErr := writer.CreateHeader(&header)
		if createErr != nil {
			writer.Close()
			output.Close()
			return createErr
		}
		sourceEntry, openErr := entry.Open()
		if openErr != nil {
			writer.Close()
			output.Close()
			return openErr
		}
		if filepath.ToSlash(entry.Name) == filepath.ToSlash(opfPath) {
			found = true
			data, readErr := io.ReadAll(io.LimitReader(sourceEntry, maximumPackageBytes+1))
			sourceEntry.Close()
			if readErr != nil || len(data) > maximumPackageBytes {
				writer.Close()
				output.Close()
				return errors.New("package metadata is too large")
			}
			updated, updateErr := updatePackage(string(data), changes)
			if updateErr != nil {
				writer.Close()
				output.Close()
				return updateErr
			}
			_, err = io.WriteString(destination, updated)
		} else {
			_, err = io.Copy(destination, sourceEntry)
			sourceEntry.Close()
		}
		if err != nil {
			writer.Close()
			output.Close()
			return err
		}
	}
	if !found {
		writer.Close()
		output.Close()
		return errors.New("package document is missing")
	}
	if err := writer.Close(); err != nil {
		output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func updatePackage(opf string, changes map[string]string) (string, error) {
	for field, value := range changes {
		value = strings.TrimSpace(value)
		if value == "" {
			return "", fmt.Errorf("%s cannot be empty", field)
		}
		switch field {
		case "title":
			if !titleElement.MatchString(opf) {
				return "", errors.New("package metadata has no dc:title")
			}
			// EPUB 3 allows a main title plus a refined subtitle. Rewriting every
			// dc:title would overwrite the subtitle and leave two identical titles.
			if len(titleElement.FindAllStringIndex(opf, -1)) != 1 {
				return "", errors.New("title repair requires exactly one title record")
			}
			opf = replaceElement(titleElement, opf, html.EscapeString(value))
		case "authors":
			if !creatorElement.MatchString(opf) {
				return "", errors.New("package metadata has no dc:creator")
			}
			// A single reviewed value deliberately updates only a single creator;
			// multi-role author edits need a richer review model.
			if len(creatorElement.FindAllStringIndex(opf, -1)) != 1 {
				return "", errors.New("author repair requires exactly one creator record")
			}
			opf = replaceElement(creatorElement, opf, html.EscapeString(value))
		case "series_index":
			replaced := false
			opf = seriesIndexTag.ReplaceAllStringFunc(opf, func(tag string) string {
				replaced = true
				if !seriesIndexContent.MatchString(tag) {
					return strings.TrimSuffix(tag, "/>") + ` content="` + html.EscapeString(value) + `"/>`
				}
				return seriesIndexContent.ReplaceAllStringFunc(tag, func(match string) string {
					parts := seriesIndexContent.FindStringSubmatch(match)
					return parts[1] + html.EscapeString(value) + parts[3]
				})
			})
			if !replaced {
				index := strings.Index(strings.ToLower(opf), "</metadata>")
				if index < 0 {
					return "", errors.New("package metadata has no metadata element")
				}
				opf = opf[:index] + `<meta name="calibre:series_index" content="` + html.EscapeString(value) + `"/>` + opf[index:]
			}
		default:
			return "", fmt.Errorf("unsupported metadata field %q", field)
		}
	}
	return opf, nil
}

func replaceElement(pattern *regexp.Regexp, source, value string) string {
	return pattern.ReplaceAllStringFunc(source, func(match string) string {
		parts := pattern.FindStringSubmatch(match)
		return parts[1] + value + parts[3]
	})
}

func packagePath(files []*zip.File) (string, error) {
	data, err := zipEntry(files, "META-INF/container.xml")
	if err != nil {
		return "", err
	}
	var container struct {
		Rootfiles []struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.Unmarshal(data, &container); err != nil || len(container.Rootfiles) == 0 {
		return "", errors.New("container has no package document")
	}
	path := filepath.ToSlash(container.Rootfiles[0].Path)
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "../") {
		return "", errors.New("invalid package path")
	}
	return path, nil
}

func validateEPUB(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	opfPath, err := packagePath(reader.File)
	if err != nil {
		return err
	}
	data, err := zipEntry(reader.File, opfPath)
	if err != nil {
		return err
	}
	var document any
	return xml.Unmarshal(data, &document)
}

func zipEntry(files []*zip.File, name string) ([]byte, error) {
	for _, entry := range files {
		if filepath.ToSlash(entry.Name) != filepath.ToSlash(name) {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		data, err := io.ReadAll(io.LimitReader(reader, maximumPackageBytes+1))
		if err != nil || len(data) > maximumPackageBytes {
			return nil, errors.New("EPUB metadata entry is too large")
		}
		return data, nil
	}
	return nil, fmt.Errorf("EPUB entry %q is missing", name)
}

func createExclusiveBackup(source, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	if err := output.Sync(); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}
