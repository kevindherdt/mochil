// Package customcolumns holds the columns a person adds to the library editor (a genre, a "to review" flag, a score)
// and the value each book has in them. It is Mochil's own data: books and their files are never touched.
package customcolumns

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

const (
	Text   = "text"
	Number = "number"
	YesNo  = "yesno"
	Tags   = "tags"

	maximumColumns = 40
	maximumValue   = 2000
	maximumLabel   = 40
)

var ErrInvalid = errors.New("invalid custom column")

// Column is one added column. IDs start with "#" like Calibre's lookup names, so they cannot clash with book fields.
type Column struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

type data struct {
	Columns []Column `json:"columns"`
	// Values is library -> book -> column id -> value. An empty value is never stored.
	Values map[string]map[string]map[string]string `json:"values"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data data
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: data{Values: map[string]map[string]map[string]string{}}}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return nil, fmt.Errorf("read custom columns: %w", err)
	}
	if s.data.Values == nil {
		s.data.Values = map[string]map[string]map[string]string{}
	}
	return s, nil
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, s.path)
}

func (s *Store) Columns() []Column {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Column{}, s.data.Columns...)
}

func (s *Store) column(id string) (Column, bool) {
	for _, column := range s.data.Columns {
		if column.ID == id {
			return column, true
		}
	}
	return Column{}, false
}

// slug turns a label into the part of an id after "#": lowercase letters and digits, words joined by "_".
func slug(label string) string {
	var out strings.Builder
	pendingGap := false
	for _, character := range strings.ToLower(label) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if pendingGap && out.Len() > 0 {
				out.WriteByte('_')
			}
			pendingGap = false
			out.WriteRune(character)
		} else {
			pendingGap = true
		}
	}
	return out.String()
}

// Add creates a column. Two labels that slug alike get distinct ids, so a label can be reused after a removal.
func (s *Store) Add(label, kind string) (Column, error) {
	label = strings.TrimSpace(label)
	name := slug(label)
	switch kind {
	case Text, Number, YesNo, Tags:
	default:
		return Column{}, fmt.Errorf("%w: unknown type", ErrInvalid)
	}
	if name == "" || len([]rune(label)) > maximumLabel {
		return Column{}, fmt.Errorf("%w: give it a name of up to %d characters", ErrInvalid, maximumLabel)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data.Columns) >= maximumColumns {
		return Column{}, fmt.Errorf("%w: at most %d columns", ErrInvalid, maximumColumns)
	}
	for _, existing := range s.data.Columns {
		if strings.EqualFold(existing.Label, label) {
			return Column{}, fmt.Errorf("%w: there is already a column with that name", ErrInvalid)
		}
	}
	id := "#" + name
	for n := 2; ; n++ {
		if _, taken := s.column(id); !taken {
			break
		}
		id = "#" + name + "_" + strconv.Itoa(n)
	}
	column := Column{ID: id, Label: label, Type: kind}
	previous := s.data
	s.data.Columns = append(append([]Column{}, s.data.Columns...), column)
	if err := s.saveLocked(); err != nil {
		s.data = previous
		return Column{}, err
	}
	return column, nil
}

// Remove deletes a column and every value in it.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.column(id); !ok {
		return os.ErrNotExist
	}
	previous := s.data
	columns := make([]Column, 0, len(s.data.Columns))
	for _, column := range s.data.Columns {
		if column.ID != id {
			columns = append(columns, column)
		}
	}
	values := map[string]map[string]map[string]string{}
	for library, books := range s.data.Values {
		for book, cells := range books {
			for column, value := range cells {
				if column == id {
					continue
				}
				if values[library] == nil {
					values[library] = map[string]map[string]string{}
				}
				if values[library][book] == nil {
					values[library][book] = map[string]string{}
				}
				values[library][book][column] = value
			}
		}
	}
	s.data = data{Columns: columns, Values: values}
	if err := s.saveLocked(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

// Normalize checks a value against its column's type and returns the form that is stored.
func Normalize(column Column, value string) (string, error) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > maximumValue {
		return "", fmt.Errorf("%w: that value is too long", ErrInvalid)
	}
	switch column.Type {
	case Number:
		if value == "" {
			return "", nil
		}
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || number != number || number > 1e12 || number < -1e12 {
			return "", fmt.Errorf("%w: %s takes a number", ErrInvalid, column.Label)
		}
		return strconv.FormatFloat(number, 'f', -1, 64), nil
	case YesNo:
		switch strings.ToLower(value) {
		case "":
			return "", nil
		case "yes", "y", "true", "1", "✓":
			return "yes", nil
		case "no", "n", "false", "0":
			return "no", nil
		}
		return "", fmt.Errorf("%w: %s takes yes or no", ErrInvalid, column.Label)
	case Tags:
		parts := []string{}
		for _, part := range strings.Split(value, ",") {
			if part = strings.TrimSpace(part); part != "" {
				parts = append(parts, part)
			}
		}
		return strings.Join(parts, ", "), nil
	}
	return value, nil
}

// Library returns every value in a library: book -> column -> value. The result is a copy.
func (s *Store) Library(library string) map[string]map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]map[string]string{}
	for book, cells := range s.data.Values[library] {
		copy := make(map[string]string, len(cells))
		for column, value := range cells {
			copy[column] = value
		}
		out[book] = copy
	}
	return out
}

// Set stores values (book -> column -> value; empty clears) and returns a function that puts every touched cell back.
func (s *Store) Set(library string, changes map[string]map[string]string) (func() error, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	type cell struct{ book, column string }
	before := map[cell]string{}
	normalized := map[cell]string{}
	for book, cells := range changes {
		for column, value := range cells {
			definition, ok := s.column(column)
			if !ok {
				return nil, fmt.Errorf("%w: there is no column %s", ErrInvalid, column)
			}
			stored, err := Normalize(definition, value)
			if err != nil {
				return nil, err
			}
			key := cell{book, column}
			normalized[key] = stored
			if _, done := before[key]; !done {
				before[key] = s.data.Values[library][book][column]
			}
		}
	}
	// write mutates the real stored maps directly (creating/removing the
	// book and library sub-maps as needed), rather than a full clone of
	// every library's every book's every column. Its cost is bounded by
	// the number of cells in `changes`, not by total store size.
	write := s.writeCellLocked(library)
	for key, stored := range normalized {
		write(key.book, key.column, stored)
	}
	if err := s.saveLocked(); err != nil {
		for key, value := range before {
			write(key.book, key.column, value)
		}
		return nil, err
	}
	return func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		write := s.writeCellLocked(library)
		// Capture the state right before undo runs so a failed persist
		// here restores exactly that, not the pre-Set state.
		preUndo := map[cell]string{}
		for key := range before {
			preUndo[key] = s.data.Values[library][key.book][key.column]
		}
		for key, value := range before {
			if _, ok := s.column(key.column); ok {
				write(key.book, key.column, value)
			}
		}
		if err := s.saveLocked(); err != nil {
			for key, value := range preUndo {
				write(key.book, key.column, value)
			}
			return err
		}
		return nil
	}, nil
}

// writeCellLocked returns a function that sets or clears one cell in
// library, creating or removing the book/library sub-maps as needed so a
// cell that did not exist before leaves no trace behind when cleared again.
// Shared by Set and its undo so both use the exact same map-mutation logic.
func (s *Store) writeCellLocked(library string) func(book, column, value string) {
	return func(book, column, value string) {
		if value == "" {
			if s.data.Values[library] == nil {
				return
			}
			delete(s.data.Values[library][book], column)
			if len(s.data.Values[library][book]) == 0 {
				delete(s.data.Values[library], book)
			}
			if len(s.data.Values[library]) == 0 {
				delete(s.data.Values, library)
			}
			return
		}
		if s.data.Values[library] == nil {
			s.data.Values[library] = map[string]map[string]string{}
		}
		if s.data.Values[library][book] == nil {
			s.data.Values[library][book] = map[string]string{}
		}
		s.data.Values[library][book][column] = value
	}
}
