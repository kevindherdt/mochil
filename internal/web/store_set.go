package web

import "mochil/internal/accounts"

// StoreSet is the mutable security state of the process: who exists, which
// devices may talk to the server, and which KOReader devices are registered.
//
// A live settings reload builds a whole new handler, and it used to open these
// stores again from disk. Two generations then held independent snapshots: an
// account disabled or a password reset through a request still being served by
// the old handler landed in the discarded snapshot, so the published server kept
// accepting the old credentials and the device tokens it had already loaded. One
// set per process, handed to each replacement, removes the second copy entirely.
type StoreSet struct {
	accounts        *accounts.Store
	androidDevices  *androidDeviceStore
	jellyfinDevices *androidDeviceStore
	devices         *koReaderDeviceStore
}

// NewStoreSet is where a process records the stores it opens, so a later
// generation can continue them instead of re-reading their files.
func NewStoreSet() *StoreSet { return &StoreSet{} }

// deviceStore returns the Android device registry to use: the live one when this
// process already has it, otherwise the one just opened, remembered for the next
// generation.
func (s *StoreSet) deviceStore(open func() (*androidDeviceStore, error), jellyfin bool) (*androidDeviceStore, error) {
	if s == nil {
		return open()
	}
	if jellyfin {
		if s.jellyfinDevices != nil {
			return s.jellyfinDevices, nil
		}
		store, err := open()
		if err == nil {
			s.jellyfinDevices = store
		}
		return store, err
	}
	if s.androidDevices != nil {
		return s.androidDevices, nil
	}
	store, err := open()
	if err == nil {
		s.androidDevices = store
	}
	return store, err
}

func (s *StoreSet) koReaderDevices(open func() (*koReaderDeviceStore, error)) (*koReaderDeviceStore, error) {
	if s == nil {
		return open()
	}
	if s.devices != nil {
		return s.devices, nil
	}
	store, err := open()
	if err == nil {
		s.devices = store
	}
	return store, err
}

func (s *StoreSet) accountStore(open func() (*accounts.Store, error)) (*accounts.Store, error) {
	if s == nil {
		return open()
	}
	if s.accounts != nil {
		return s.accounts, nil
	}
	store, err := open()
	if err == nil {
		s.accounts = store
	}
	return store, err
}

// adopt records a store this generation opened through a path the helpers above
// do not cover, such as the first-run page creating the administrator.
func (s *StoreSet) adoptAccounts(store *accounts.Store) {
	if s != nil {
		s.accounts = store
	}
}
