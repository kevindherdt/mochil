package audio

// ApplyAnalysis revision-checks a complete result batch and persists it with a
// single atomic cache replacement. Embedded ReplayGain values always win.
func (s *Store) ApplyAnalysis(results map[string]Analysis) (int, error) {
	if len(results) == 0 {
		return 0, nil
	}
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	items := s.Items()
	updated := 0
	for index := range items {
		result, ok := results[items[index].ID]
		if !ok || result.Revision != items[index].Revision {
			continue
		}
		changed := false
		if items[index].ReplayTrack == 0 {
			items[index].ReplayTrack = result.GainDB
			changed = true
		}
		if items[index].PeakTrack == 0 {
			items[index].PeakTrack = result.Peak
			changed = true
		}
		if changed {
			items[index].AnalysisRevision = result.Revision
			updated++
		}
	}
	if updated == 0 {
		return 0, nil
	}
	if err := s.save(items); err != nil {
		return 0, err
	}
	s.publish(items)
	return updated, nil
}

func (s *Store) MissingAnalysis() []Item {
	items := s.Items()
	out := make([]Item, 0)
	for _, item := range items {
		if (item.ReplayTrack == 0 || item.PeakTrack == 0) && item.AnalysisRevision != item.Revision {
			out = append(out, item)
		}
	}
	return out
}
