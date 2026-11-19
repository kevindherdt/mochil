package audio

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mochil/internal/jobs"
)

type AnalysisQueue struct {
	analyzer Analyzer
	pool     *jobs.Pool
}

func NewAnalysisQueue(analyzer Analyzer, workers int) *AnalysisQueue {
	if analyzer == nil {
		return nil
	}
	if workers < 1 || workers > 2 {
		workers = 1
	}
	return &AnalysisQueue{analyzer: analyzer, pool: jobs.New(workers, 8)}
}

func (q *AnalysisQueue) Submit(store *Store) (string, error) {
	if q == nil || q.pool == nil || store == nil {
		return "", errors.New("audio analysis is unavailable")
	}
	key := "audio-analysis:" + store.libraryID
	items := store.MissingAnalysis()
	total := int64(len(items))
	err := q.pool.Submit(key, func(ctx context.Context) error {
		results := make(map[string]Analysis, len(items))
		failed := 0
		jobs.Report(ctx, 0, total, "Analyzing loudness")
		for index, item := range items {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			file, opened, err := store.Open(item.ID)
			if err == nil {
				var result Analysis
				itemContext, cancel := context.WithTimeout(ctx, 30*time.Minute)
				result, err = q.analyzer.Analyze(itemContext, file)
				cancel()
				if err == nil {
					result.Revision = opened.Revision
					results[item.ID] = result
				}
			}
			if err != nil {
				failed++
			}
			jobs.Report(ctx, int64(index+1), total, fmt.Sprintf("%d analyzed · %d failed", len(results), failed))
		}
		if _, err := store.ApplyAnalysis(results); err != nil {
			return err
		}
		if len(items) > 0 && len(results) == 0 {
			return errors.New("no audio items could be analyzed")
		}
		return nil
	})
	return key, err
}
func (q *AnalysisQueue) Jobs() []jobs.State {
	if q == nil || q.pool == nil {
		return nil
	}
	return q.pool.Snapshot()
}
func (q *AnalysisQueue) Close() {
	if q != nil && q.pool != nil {
		q.pool.Close()
	}
}
