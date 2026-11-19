package ingest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const MaximumBatchPreviewEntries = 100

type PreviewIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PreviewEntry struct {
	SourcePath string         `json:"source_path"`
	TargetPath string         `json:"target_path"`
	Size       int64          `json:"size,omitempty"`
	Ready      bool           `json:"ready"`
	Issues     []PreviewIssue `json:"issues,omitempty"`
}

type BatchPreview struct {
	Entries   []PreviewEntry `json:"entries"`
	Ready     bool           `json:"ready"`
	Conflicts int            `json:"conflicts"`
}

// PreviewBatch performs the same confinement and no-clobber checks needed by
// planning without creating a journal transaction or touching either root.
func (e *Executor) PreviewBatch(entries []PlanEntry) (BatchPreview, error) {
	if len(entries) == 0 || len(entries) > MaximumBatchPreviewEntries {
		return BatchPreview{}, errors.New("managed ingest preview requires between 1 and 100 entries")
	}
	result := BatchPreview{Entries: make([]PreviewEntry, len(entries)), Ready: true}
	sources, targets := make(map[string][]int), make(map[string][]int)
	for index, input := range entries {
		entry := PreviewEntry{SourcePath: input.SourcePath, TargetPath: input.TargetPath, Ready: true}
		if !safeJournalPath(input.SourcePath) || !safeJournalPath(input.TargetPath) || strings.HasPrefix(input.TargetPath, ".ingest/") {
			entry.add("unsafe_path", "Keep both paths relative and inside their configured folders.")
		} else {
			_, info, err := e.resolveSource(input.SourcePath)
			if err != nil {
				entry.add("source_unavailable", "The incoming file is missing, symbolic, or no longer a regular file.")
			} else if info.Size() > e.maxFileBytes {
				entry.add("source_too_large", "The incoming file exceeds the configured file-size limit.")
			} else {
				entry.Size = info.Size()
			}
			target := filepath.Join(e.targetRoot, filepath.FromSlash(input.TargetPath))
			if _, err := os.Lstat(target); err == nil {
				entry.add("target_exists", "A managed file already uses this destination.")
			} else if !errors.Is(err, os.ErrNotExist) {
				entry.add("target_unavailable", "The destination could not be checked safely.")
			}
			sources[portablePath(input.SourcePath)] = append(sources[portablePath(input.SourcePath)], index)
			targets[portablePath(input.TargetPath)] = append(targets[portablePath(input.TargetPath)], index)
		}
		result.Entries[index] = entry
	}
	for _, indexes := range sources {
		if len(indexes) > 1 {
			for _, index := range indexes {
				result.Entries[index].add("duplicate_source", "This incoming file appears more than once in the batch.")
			}
		}
	}
	for _, indexes := range targets {
		if len(indexes) > 1 {
			for _, index := range indexes {
				result.Entries[index].add("duplicate_target", "Multiple files would use this managed destination.")
			}
		}
	}
	for index := range result.Entries {
		result.Entries[index].Ready = len(result.Entries[index].Issues) == 0
		if !result.Entries[index].Ready {
			result.Ready = false
			result.Conflicts++
		}
	}
	return result, nil
}

func (e *PreviewEntry) add(code, message string) {
	for _, issue := range e.Issues {
		if issue.Code == code {
			return
		}
	}
	e.Issues = append(e.Issues, PreviewIssue{Code: code, Message: message})
}

func portablePath(value string) string {
	return strings.ToLower(filepath.ToSlash(filepath.Clean(filepath.FromSlash(value))))
}
