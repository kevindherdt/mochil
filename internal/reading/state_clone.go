package reading

func clonePosition(position Position) Position {
	position.KoSyncDocuments = append([]string(nil), position.KoSyncDocuments...)
	if position.WebAnchor != nil {
		anchor := *position.WebAnchor
		position.WebAnchor = &anchor
	}
	if position.Locators != nil {
		locators := make(map[string]string, len(position.Locators))
		for kind, locator := range position.Locators {
			locators[kind] = locator
		}
		position.Locators = locators
	}
	return position
}

func clonePositions(positions []Position) []Position {
	out := make([]Position, len(positions))
	for index := range positions {
		out[index] = clonePosition(positions[index])
	}
	return out
}

// ensureAccountLocked returns the account's position map, creating it if
// absent. A fresh map is stored immediately, so every later mutation in the
// same call sees it.
func (s *Store) ensureAccountLocked(account string) map[string]Position {
	positions := s.data.Accounts[account]
	if positions != nil {
		return positions
	}
	positions = map[string]Position{}
	s.data.Accounts[account] = positions
	return positions
}

func cloneStoreData(data storeData) storeData {
	clone := storeData{
		Version:  data.Version,
		Accounts: make(map[string]map[string]Position, len(data.Accounts)),
		History:  make(map[string]map[string][]Position, len(data.History)),
	}
	for account, positions := range data.Accounts {
		copyPositions := make(map[string]Position, len(positions))
		for key, position := range positions {
			copyPositions[key] = clonePosition(position)
		}
		clone.Accounts[account] = copyPositions
	}
	for account, histories := range data.History {
		copyHistories := make(map[string][]Position, len(histories))
		for key, history := range histories {
			copyHistory := make([]Position, len(history))
			for index, position := range history {
				copyHistory[index] = clonePosition(position)
			}
			copyHistories[key] = copyHistory
		}
		clone.History[account] = copyHistories
	}
	return clone
}
