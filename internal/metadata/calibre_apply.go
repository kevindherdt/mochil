package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maximumCalibreOutput = 16 << 20

// CalibreRunner runs calibredb without involving a shell. Keeping this small
// makes the write path testable without ever opening a real Calibre library.
type CalibreRunner interface {
	Run(context.Context, ...string) ([]byte, error)
}

type CalibreCommand struct{ Path string }

func (c CalibreCommand) Run(ctx context.Context, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, c.Path, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	output := stdout.Bytes()
	if len(output) > maximumCalibreOutput {
		return nil, errors.New("calibredb returned too much output")
	}
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(string(output))
		}
		if len(detail) > 1000 {
			detail = detail[:1000]
		}
		if detail != "" {
			return nil, fmt.Errorf("calibredb: %s", detail)
		}
		return nil, fmt.Errorf("calibredb: %w", err)
	}
	return output, nil
}

func FindCalibreCommand(path string) (CalibreCommand, error) {
	if strings.TrimSpace(path) == "" {
		path = "calibredb"
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return CalibreCommand{}, errors.New("calibredb is not installed in the Mochil container")
	}
	return CalibreCommand{Path: resolved}, nil
}

type CalibreTarget struct {
	BookPath string
	Backup   string
	Changes  map[string]string
}

type resolvedCalibreTarget struct {
	CalibreTarget
	id string
}

// ApplyCalibreBatch owns the complete confirmed Calibre transaction. It
// resolves exact record IDs, captures durable OPF backups, applies and verifies
// every record, refreshes sidecars, and returns a compensating rollback for the
// outer review-job transaction.
func ApplyCalibreBatch(ctx context.Context, runner CalibreRunner, library string, targets []CalibreTarget) (func() error, error) {
	if len(targets) == 0 {
		return nil, errors.New("the review has no accepted changes")
	}
	ordered := append([]CalibreTarget(nil), targets...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].BookPath < ordered[j].BookPath })
	for _, target := range ordered {
		if err := PreflightCalibre(target); err != nil {
			return nil, fmt.Errorf("preflight failed for %s: %w", filepath.Base(target.BookPath), err)
		}
	}
	transactionCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	ids, err := ResolveCalibreIDs(transactionCtx, runner, library, ordered)
	if err != nil {
		return nil, fmt.Errorf("Calibre preflight failed: %w", err)
	}
	resolved := make([]resolvedCalibreTarget, 0, len(ordered))
	for _, target := range ordered {
		resolvedTarget := resolvedCalibreTarget{CalibreTarget: target, id: ids[target.BookPath]}
		if err := BackupCalibre(transactionCtx, runner, library, resolvedTarget.id, target.Backup); err != nil {
			return nil, fmt.Errorf("could not back up %s: %w", filepath.Base(target.BookPath), err)
		}
		resolved = append(resolved, resolvedTarget)
	}
	applied := make([]resolvedCalibreTarget, 0, len(resolved))
	rollback := func() error {
		rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer rollbackCancel()
		var failures []string
		for index := len(applied) - 1; index >= 0; index-- {
			target := applied[index]
			if err := RestoreCalibre(rollbackCtx, runner, library, target.id, target.Backup); err != nil {
				failures = append(failures, err.Error())
			}
		}
		if len(failures) != 0 {
			return errors.New(strings.Join(failures, "; "))
		}
		if len(applied) != 0 {
			if err := FlushCalibreSidecars(rollbackCtx, runner, library); err != nil {
				return fmt.Errorf("restore sidecar refresh: %w", err)
			}
		}
		return nil
	}
	for _, target := range resolved {
		// Include the target before mutation because set_metadata may succeed even
		// when the following read-back verification fails.
		applied = append(applied, target)
		if err := ApplyCalibre(transactionCtx, runner, library, target.id, target.Changes); err != nil {
			if rollbackErr := rollback(); rollbackErr != nil {
				return nil, &TransactionError{Err: fmt.Errorf("Calibre apply failed and rollback needs attention: %w", rollbackErr)}
			}
			return nil, fmt.Errorf("Calibre apply failed; earlier changes were restored: %w", err)
		}
	}
	if err := FlushCalibreSidecars(transactionCtx, runner, library); err != nil {
		if rollbackErr := rollback(); rollbackErr != nil {
			return nil, &TransactionError{Err: fmt.Errorf("Calibre sidecar refresh failed and rollback needs attention: %w", rollbackErr)}
		}
		return nil, fmt.Errorf("Calibre sidecar refresh failed; changes were restored: %w", err)
	}
	return rollback, nil
}

type calibreListBook struct {
	ID      json.RawMessage `json:"id"`
	Formats any             `json:"formats"`
}

func calibreArgs(library string, command ...string) []string {
	return append([]string{"--with-library", library}, command...)
}

// ResolveCalibreIDs maps Mochil publication paths to Calibre IDs using the
// exact format paths reported by Calibre itself. It never guesses from titles
// or the conventional "Title (id)" folder name.
func ResolveCalibreIDs(ctx context.Context, runner CalibreRunner, library string, targets []CalibreTarget) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	output, err := runner.Run(ctx, calibreArgs(library, "list", "--fields", "id,formats", "--for-machine")...)
	if err != nil {
		return nil, err
	}
	var rows []calibreListBook
	if err := json.Unmarshal(output, &rows); err != nil {
		return nil, errors.New("calibredb returned invalid book-list JSON")
	}
	wanted := make(map[string]string, len(targets))
	for _, target := range targets {
		absolute, err := filepath.Abs(target.BookPath)
		if err != nil {
			return nil, err
		}
		wanted[filepath.Clean(absolute)] = target.BookPath
	}
	resolved := make(map[string]string, len(targets))
	for _, row := range rows {
		id, err := calibreID(row.ID)
		if err != nil {
			continue
		}
		for _, format := range calibreFormats(row.Formats) {
			absolute := format
			if !filepath.IsAbs(absolute) {
				absolute = filepath.Join(library, absolute)
			}
			if original, ok := wanted[filepath.Clean(absolute)]; ok {
				if prior, duplicate := resolved[original]; duplicate && prior != id {
					return nil, fmt.Errorf("publication %q matched more than one Calibre record", original)
				}
				resolved[original] = id
			}
		}
	}
	for _, target := range targets {
		if resolved[target.BookPath] == "" {
			return nil, fmt.Errorf("Calibre could not identify publication %q", filepath.Base(target.BookPath))
		}
	}
	return resolved, nil
}

func calibreID(raw json.RawMessage) (string, error) {
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		if _, err := strconv.ParseInt(number.String(), 10, 64); err == nil {
			return number.String(), nil
		}
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if _, err := strconv.ParseInt(text, 10, 64); err == nil {
			return text, nil
		}
	}
	return "", errors.New("invalid Calibre ID")
}

func calibreFormats(value any) []string {
	switch formats := value.(type) {
	case string:
		return strings.Split(formats, ",")
	case []any:
		out := make([]string, 0, len(formats))
		for _, value := range formats {
			if text, ok := value.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func PreflightCalibre(target CalibreTarget) error {
	if target.BookPath == "" || target.Backup == "" {
		return errors.New("Calibre apply target is incomplete")
	}
	if info, err := os.Stat(target.Backup); err == nil {
		if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maximumCalibreOutput {
			return errors.New("existing Calibre backup is invalid")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if len(target.Changes) == 0 {
		return errors.New("Calibre apply target has no changes")
	}
	for field, value := range target.Changes {
		switch field {
		case "title", "authors", "series_index":
		default:
			return fmt.Errorf("unsupported Calibre metadata field %q", field)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("Calibre metadata field %q cannot be empty", field)
		}
	}
	return nil
}

func BackupCalibre(ctx context.Context, runner CalibreRunner, library, id, destination string) error {
	if existing, err := os.ReadFile(destination); err == nil {
		if err := validateCalibreOPF(existing); err != nil {
			return fmt.Errorf("existing Calibre backup is invalid: %w", err)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	output, err := runner.Run(ctx, calibreArgs(library, "show_metadata", "--as-opf", id)...)
	if err != nil {
		return err
	}
	if err := validateCalibreOPF(output); err != nil {
		return fmt.Errorf("invalid Calibre backup: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return err
	}
	// Temp file → Sync → Rename, into the same directory: written straight to its final name, a crash or a full disk
	// left a truncated OPF at the destination, and the next attempt saw the file, failed validation and returned
	// "existing Calibre backup is invalid" for good. The O_EXCL check below still reads the final name, so an existing
	// backup is never replaced.
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".backup-*.opf")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(output); err != nil {
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
	if err := os.Chmod(temporary.Name(), 0o600); err != nil {
		return err
	}
	if _, err := os.Lstat(destination); err == nil {
		return os.ErrExist
	}
	return os.Rename(temporary.Name(), destination)
}

func ApplyCalibre(ctx context.Context, runner CalibreRunner, library, id string, changes map[string]string) error {
	args := calibreArgs(library, "set_metadata", id)
	for _, field := range []string{"title", "authors", "series_index"} {
		if value, ok := changes[field]; ok {
			args = append(args, "--field", field+":"+value)
		}
	}
	if _, err := runner.Run(ctx, args...); err != nil {
		return err
	}
	output, err := runner.Run(ctx, calibreArgs(library, "show_metadata", "--as-opf", id)...)
	if err != nil {
		return err
	}
	return verifyCalibreOPF(output, changes)
}

func RestoreCalibre(ctx context.Context, runner CalibreRunner, library, id, backup string) error {
	_, err := runner.Run(ctx, calibreArgs(library, "set_metadata", id, backup)...)
	return err
}

func FlushCalibreSidecars(ctx context.Context, runner CalibreRunner, library string) error {
	_, err := runner.Run(ctx, calibreArgs(library, "backup_metadata")...)
	return err
}

type calibreOPF struct {
	Metadata struct {
		Titles   []string `xml:"title"`
		Creators []string `xml:"creator"`
		Metas    []struct {
			Name    string `xml:"name,attr"`
			Content string `xml:"content,attr"`
		} `xml:"meta"`
	} `xml:"metadata"`
}

func parseCalibreOPF(data []byte) (calibreOPF, error) {
	var opf calibreOPF
	if len(data) == 0 || len(data) > maximumCalibreOutput {
		return opf, errors.New("OPF output has an invalid size")
	}
	if err := xml.Unmarshal(data, &opf); err != nil {
		return opf, err
	}
	return opf, nil
}

func validateCalibreOPF(data []byte) error {
	_, err := parseCalibreOPF(data)
	return err
}

func verifyCalibreOPF(data []byte, changes map[string]string) error {
	opf, err := parseCalibreOPF(data)
	if err != nil {
		return fmt.Errorf("Calibre verification failed: %w", err)
	}
	actual := map[string]string{}
	if len(opf.Metadata.Titles) > 0 {
		actual["title"] = strings.TrimSpace(opf.Metadata.Titles[0])
	}
	if len(opf.Metadata.Creators) > 0 {
		actual["authors"] = strings.Join(opf.Metadata.Creators, " & ")
	}
	for _, meta := range opf.Metadata.Metas {
		if strings.EqualFold(meta.Name, "calibre:series_index") {
			actual["series_index"] = strings.TrimSpace(meta.Content)
		}
	}
	for field, expected := range changes {
		if field == "series_index" {
			want, wantErr := strconv.ParseFloat(strings.TrimSpace(expected), 64)
			got, gotErr := strconv.ParseFloat(actual[field], 64)
			if wantErr == nil && gotErr == nil && want == got {
				continue
			}
		} else if strings.TrimSpace(actual[field]) == strings.TrimSpace(expected) {
			continue
		}
		return fmt.Errorf("Calibre did not persist the requested %s", field)
	}
	return nil
}
