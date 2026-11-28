package web

import "mochil/internal/accounts"

// withAccounts seeds the account store the way production publishes it -- through the atomic -- so a test can never observe
// a server whose account store was assigned behind a reader's back.
func withAccounts(store *accounts.Store, configure func(*multiServer)) *multiServer {
	server := &multiServer{}
	if configure != nil {
		configure(server)
	}
	server.accountsStoreField.Store(store)
	return server
}
