package web

func cloneCollectionData(data collectionData) collectionData {
	clone := collectionData{Version: data.Version, Accounts: make(map[string][]savedCollection, len(data.Accounts))}
	for account, collections := range data.Accounts {
		copyCollections := make([]savedCollection, len(collections))
		for index, collection := range collections {
			collection.Members = append([]collectionMember(nil), collection.Members...)
			copyCollections[index] = collection
		}
		clone.Accounts[account] = copyCollections
	}
	return clone
}
