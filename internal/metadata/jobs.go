package metadata

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"mochil/internal/catalog"
	"mochil/internal/ingest"
	"mochil/internal/statefile"
)

const jobFileVersion = 1
const maximumJobStateBytes int64 = 128 << 20

var (
	volumeMarker         = regexp.MustCompile(`(?i)(?:\bvol(?:ume)?\.?|\bv|#)\s*([0-9]+(?:\.[0-9]+)?)`)
	bookMarker           = regexp.MustCompile(`(?i)\bbook\s*([0-9]+(?:\.[0-9]+)?)`)
	partMarker           = regexp.MustCompile(`(?i)\bpart\s*([0-9]+(?:\.[0-9]+)?)`)
	volumeRangeRemainder = regexp.MustCompile(`^\s*[-–—]\s*[0-9]`)
)

type Proposal struct {
	Field      string `json:"field"`
	Current    string `json:"current,omitempty"`
	Proposed   string `json:"proposed,omitempty"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

type ReviewItem struct {
	Issue
	Proposal      *Proposal `json:"proposal,omitempty"`
	Decision      string    `json:"decision,omitempty"`
	DecisionValue string    `json:"decision_value,omitempty"`
}

type Job struct {
	ID                 string         `json:"id"`
	LibraryID          string         `json:"library_id"`
	LibraryName        string         `json:"library_name"`
	Status             string         `json:"status"`
	CreatedAt          time.Time      `json:"created_at"`
	AppliedAt          *time.Time     `json:"applied_at,omitempty"`
	BooksInspected     int            `json:"books_inspected"`
	IssueCount         int            `json:"issue_count"`
	InheritedDecisions int            `json:"inherited_decisions,omitempty"`
	Counts             map[string]int `json:"counts"`
	Items              []ReviewItem   `json:"items"`
}

type jobFile struct {
	Version int   `json:"version"`
	Jobs    []Job `json:"jobs"`
}

type JobStore struct {
	path string
	mu   sync.RWMutex
	jobs []Job
}

type Decision struct {
	BookID   string `json:"book_id"`
	Code     string `json:"code"`
	Decision string `json:"decision"`
	Value    string `json:"value"`
}

func OpenJobStore(path string) (*JobStore, error) {
	store := &JobStore{path: path}
	if path == "" {
		return store, nil
	}
	var file jobFile
	err := statefile.DecodeJSON(path, maximumJobStateBytes, &file)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("decode metadata review jobs: %w", err)
	}
	if file.Version != jobFileVersion {
		return nil, fmt.Errorf("unsupported metadata review job version %d", file.Version)
	}
	if len(file.Jobs) > 50 {
		return nil, errors.New("metadata review job store contains too many jobs")
	}
	store.jobs = file.Jobs
	return store, nil
}

func (s *JobStore) Create(libraryID, libraryName string, books []catalog.Book) (Job, error) {
	report := Inspect(books)
	seriesAuthors := consistentSeriesAuthors(books)
	job := Job{ID: newJobID(), LibraryID: libraryID, LibraryName: libraryName, Status: "review", CreatedAt: time.Now().UTC(), BooksInspected: report.BooksInspected, IssueCount: report.IssueCount, Counts: report.Counts, Items: make([]ReviewItem, 0, len(report.Issues))}
	for _, issue := range report.Issues {
		job.Items = append(job.Items, ReviewItem{Issue: issue, Proposal: propose(issue, seriesAuthors)})
	}
	s.mu.Lock()
	job.InheritedDecisions = s.inheritDecisionsLocked(&job)
	job.Status = jobStatus(job.Items)
	previous := s.jobs
	s.jobs = append([]Job{job}, s.jobs...)
	if len(s.jobs) > 50 {
		s.jobs = s.jobs[:50]
	}
	err := s.saveLocked()
	if err != nil {
		s.jobs = previous
	}
	s.mu.Unlock()
	return job, err
}

// inheritDecisionsLocked carries the newest explicit choice into a new
// inspection. A pending choice is a tombstone: it deliberately restores the
// issue to review and prevents an older accepted/ignored choice resurfacing.
func (s *JobStore) inheritDecisionsLocked(job *Job) int {
	inherited := 0
	for itemIndex := range job.Items {
		item := &job.Items[itemIndex]
		matched := false
		for _, previous := range s.jobs {
			if previous.LibraryID != job.LibraryID {
				continue
			}
			for _, old := range previous.Items {
				if old.BookID != item.BookID || old.Code != item.Code {
					continue
				}
				matched = true
				if old.Decision == "accepted" || old.Decision == "ignored" {
					item.Decision = old.Decision
					item.DecisionValue = old.DecisionValue
					inherited++
				}
				break
			}
			if matched {
				break
			}
		}
	}
	return inherited
}

func (s *JobStore) List(libraryID string) []Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		if libraryID == "" || job.LibraryID == libraryID {
			copyJob := job
			copyJob.Items = nil
			result = append(result, copyJob)
		}
	}
	return result
}

func (s *JobStore) Get(id string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, job := range s.jobs {
		if job.ID == id {
			return cloneJob(job), true
		}
	}
	return Job{}, false
}

func (s *JobStore) Decide(jobID, bookID, code, decision, value string) (Job, error) {
	return s.DecideMany(jobID, []Decision{{BookID: bookID, Code: code, Decision: decision, Value: value}})
}

func (s *JobStore) DecideMany(jobID string, decisions []Decision) (Job, error) {
	if len(decisions) == 0 || len(decisions) > 500 {
		return Job{}, errors.New("between 1 and 500 review decisions are required")
	}
	for index := range decisions {
		decisions[index].Decision = strings.TrimSpace(decisions[index].Decision)
		decisions[index].Value = strings.TrimSpace(decisions[index].Value)
		if decisions[index].Decision != "accepted" && decisions[index].Decision != "ignored" && decisions[index].Decision != "pending" {
			return Job{}, errors.New("decision must be accepted, ignored, or pending")
		}
		if decisions[index].Decision == "accepted" && decisions[index].Value == "" {
			return Job{}, errors.New("an accepted decision requires a value")
		}
		if len(decisions[index].Value) > 500 {
			return Job{}, errors.New("decision value cannot exceed 500 characters")
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for jobIndex := range s.jobs {
		if s.jobs[jobIndex].ID != jobID {
			continue
		}
		if s.jobs[jobIndex].Status == "applied" {
			return Job{}, errors.New("an applied review cannot be changed")
		}
		original := cloneJob(s.jobs[jobIndex])
		for _, update := range decisions {
			found := false
			for itemIndex := range s.jobs[jobIndex].Items {
				item := &s.jobs[jobIndex].Items[itemIndex]
				if item.BookID != update.BookID || item.Code != update.Code {
					continue
				}
				item.Decision, item.DecisionValue = update.Decision, update.Value
				if update.Decision != "accepted" {
					item.DecisionValue = ""
				}
				found = true
				break
			}
			if !found {
				s.jobs[jobIndex] = original
				return Job{}, errors.New("review item not found")
			}
		}
		s.jobs[jobIndex].Status = jobStatus(s.jobs[jobIndex].Items)
		if err := s.saveLocked(); err != nil {
			s.jobs[jobIndex] = original
			return Job{}, err
		}
		return cloneJob(s.jobs[jobIndex]), nil
	}
	return Job{}, errors.New("review job not found")
}

func (s *JobStore) MarkApplied(jobID string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.jobs {
		if s.jobs[index].ID != jobID {
			continue
		}
		if s.jobs[index].Status != "resolved" {
			return Job{}, errors.New("review job is not resolved")
		}
		now := time.Now().UTC()
		s.jobs[index].Status, s.jobs[index].AppliedAt = "applied", &now
		if err := s.saveLocked(); err != nil {
			s.jobs[index].Status, s.jobs[index].AppliedAt = "resolved", nil
			return Job{}, err
		}
		return cloneJob(s.jobs[index]), nil
	}
	return Job{}, errors.New("review job not found")
}

func jobStatus(items []ReviewItem) string {
	for _, item := range items {
		if item.Decision == "" || item.Decision == "pending" {
			return "review"
		}
	}
	return "resolved"
}

func cloneJob(job Job) Job {
	job.Items = append([]ReviewItem(nil), job.Items...)
	job.Counts = make(map[string]int, len(job.Counts))
	for code, count := range job.Counts {
		job.Counts[code] = count
	}
	return job
}

func (s *JobStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(jobFile{Version: jobFileVersion, Jobs: s.jobs}, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".metadata-reviews-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
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
	return os.Rename(temporaryName, s.path)
}

func propose(issue Issue, seriesAuthors map[string]string) *Proposal {
	switch issue.Code {
	case "missing_author":
		if author := seriesAuthors[normalizedSeries(issue.Series)]; author != "" {
			return &Proposal{Field: "authors", Current: "Unknown", Proposed: author, Source: "consistent series siblings", Confidence: "high"}
		}
		inferred := ingest.InferPath(ingest.Publication, issue.Path)
		if len(inferred.Authors) == 1 && strings.TrimSpace(inferred.Authors[0]) != "" {
			return &Proposal{Field: "authors", Current: "Unknown", Proposed: inferred.Authors[0], Source: "author folder", Confidence: "medium"}
		}
	case "missing_title":
		inferred := ingest.InferPath(ingest.Publication, issue.Path)
		name := strings.TrimSpace(inferred.Title)
		if name != "" {
			return &Proposal{Field: "title", Current: issue.Title, Proposed: name, Source: "structured filename", Confidence: "medium"}
		}
	case "missing_series_index", "duplicate_series_index":
		if number, confidence, ok := titleVolumeProposal(issue.Title); ok {
			proposed := strconv.FormatFloat(number, 'f', -1, 64)
			current := "Not set"
			if issue.SeriesIndex != nil {
				current = strconv.FormatFloat(*issue.SeriesIndex, 'f', -1, 64)
				if number == *issue.SeriesIndex {
					return nil
				}
			}
			return &Proposal{Field: "series_index", Current: current, Proposed: proposed, Source: "title volume marker", Confidence: confidence}
		}
	}
	return nil
}

func normalizedSeries(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

// consistentSeriesAuthors returns a suggestion only when all publications in
// a series that have usable authors agree. A majority is deliberately not
// enough: contributor changes and parallel editions require human review.
func consistentSeriesAuthors(books []catalog.Book) map[string]string {
	type consensus struct {
		display   string
		signature string
		conflict  bool
	}
	bySeries := make(map[string]consensus)
	for _, book := range books {
		series := normalizedSeries(book.Series)
		if series == "" || len(book.Authors) == 0 || allBlank(book.Authors) || allUnknown(book.Authors) {
			continue
		}
		authors := make([]string, 0, len(book.Authors))
		for _, author := range book.Authors {
			if value := strings.Join(strings.Fields(strings.TrimSpace(author)), " "); value != "" {
				authors = append(authors, value)
			}
		}
		if len(authors) == 0 {
			continue
		}
		display := strings.Join(authors, " & ")
		signature := strings.ToLower(display)
		current, exists := bySeries[series]
		if !exists {
			bySeries[series] = consensus{display: display, signature: signature}
			continue
		}
		if current.signature != signature {
			current.conflict = true
			bySeries[series] = current
		}
	}
	result := make(map[string]string)
	for series, authors := range bySeries {
		if !authors.conflict {
			result[series] = authors.display
		}
	}
	return result
}

func titleVolumeNumber(title string) (float64, bool) {
	number, _, ok := titleVolumeProposal(title)
	return number, ok
}

func titleVolumeProposal(title string) (float64, string, bool) {
	// Explicit volume labels are more specific than a franchise "Part" label
	// (for example, "Part 1 ... Volume 02").
	markers := []struct {
		pattern    *regexp.Regexp
		confidence string
	}{{volumeMarker, "high"}, {bookMarker, "high"}, {partMarker, "medium"}}
	for _, marker := range markers {
		match := marker.pattern.FindStringSubmatch(title)
		if len(match) != 2 {
			continue
		}
		full := marker.pattern.FindStringIndex(title)
		if full != nil && volumeRangeRemainder.MatchString(title[full[1]:]) {
			return 0, "", false
		}
		number, err := strconv.ParseFloat(match[1], 64)
		return number, marker.confidence, err == nil
	}
	return 0, "", false
}

func newJobID() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
