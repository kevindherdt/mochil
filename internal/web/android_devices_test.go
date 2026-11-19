package web

import (
	"os"
	"testing"
)

func TestAndroidDeviceTokenIsPersistentHashedAndBounded(t *testing.T) {
	path := t.TempDir() + "/android-devices.json"
	store, err := newAndroidDeviceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	var firstToken string
	for index := range androidMaximumDevices + 1 {
		token, deviceID, err := store.issue("reader", "Android device")
		if err != nil || token == "" || deviceID == "" {
			t.Fatalf("index=%d token=%q device=%q err=%v", index, token, deviceID, err)
		}
		if index == 0 {
			firstToken = token
		}
	}
	if _, _, ok := store.authenticate(firstToken); ok {
		t.Fatal("oldest device was not evicted at the account bound")
	}
	token, deviceID, err := store.issue("reader", "Pixel")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := newAndroidDeviceStore(path)
	if err != nil {
		t.Fatal(err)
	}
	account, authenticatedID, ok := reloaded.authenticate(token)
	if !ok || account != "reader" || authenticatedID != deviceID {
		t.Fatalf("account=%q device=%q authenticated=%v", account, authenticatedID, ok)
	}
}

func TestAndroidDeviceCanRevokeItself(t *testing.T) {
	store, err := newAndroidDeviceStore(t.TempDir() + "/android-devices.json")
	if err != nil {
		t.Fatal(err)
	}
	token, deviceID, err := store.issue("reader", "Android device")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.revoke("reader", deviceID); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := store.authenticate(token); ok {
		t.Fatal("revoked token still authenticates")
	}
}

func TestFailedRevocationKeepsEveryDeviceInOrder(t *testing.T) {
	for _, target := range []int{0, 1, 2} {
		store, err := newAndroidDeviceStore(t.TempDir() + "/android-devices.json")
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, name := range []string{"A", "B", "C"} {
			_, id, err := store.issue("reader", name)
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		blocker := t.TempDir() + "/file"
		if err := os.WriteFile(blocker, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		store.path = blocker + "/android-devices.json" // the parent is a file, so saving fails
		if err := store.revoke("reader", ids[target]); err == nil {
			t.Fatal("revocation reported success although saving failed")
		}
		got := store.devices.Accounts["reader"]
		if len(got) != 3 || got[0].ID != ids[0] || got[1].ID != ids[1] || got[2].ID != ids[2] {
			t.Fatalf("target %d: devices after failed revoke = %+v", target, got)
		}
	}
}
