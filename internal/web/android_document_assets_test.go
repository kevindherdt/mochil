package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"mochil/internal/document"
)

func TestAndroidVaultPDFAttachmentRequiresDeviceAndServesExactBytes(t *testing.T) {
	root := t.TempDir()
	pdf := []byte("%PDF-1.7\nfixture bytes\n%%EOF\n")
	if err := os.WriteFile(filepath.Join(root, "Manual.pdf"), pdf, 0600); err != nil {
		t.Fatal(err)
	}
	documents := document.NewVaultStore("vault", root, "")
	if _, err := documents.Scan(); err != nil {
		t.Fatal(err)
	}
	devices, err := newAndroidDeviceStore(filepath.Join(t.TempDir(), "devices.json"))
	if err != nil {
		t.Fatal(err)
	}
	m := multiServer{cfg: MultiConfig{Username: "reader"}, androidDevices: devices,
		libraries: []Library{{ID: "vault", Kind: "vault", Documents: documents}}}
	token, deviceID, err := devices.issue(m.readingAccount(nil), "Fixture phone")
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/android/v1/document-assets/vault/" + documents.Items()[0].ID + "/Manual.pdf"
	request := func(method, address, credential, byteRange string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, address, nil)
		if credential != "" {
			r.Header.Set("Authorization", "Bearer "+credential)
		}
		if byteRange != "" {
			r.Header.Set("Range", byteRange)
		}
		response := httptest.NewRecorder()
		m.androidAPI(response, r)
		return response
	}
	if response := request(http.MethodGet, path, "", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", response.Code)
	}
	response := request(http.MethodGet, path, token, "")
	if response.Code != http.StatusOK || response.Body.String() != string(pdf) || response.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("PDF response status=%d type=%q body=%q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	if response.Header().Get("Content-Length") != strconv.Itoa(len(pdf)) || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing expected PDF length/type safeguards")
	}
	if head := request(http.MethodHead, path, token, ""); head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD status=%d bytes=%d", head.Code, head.Body.Len())
	}
	if partial := request(http.MethodGet, path, token, "bytes=0-4"); partial.Code != http.StatusPartialContent || partial.Body.String() != "%PDF-" {
		t.Fatalf("range status=%d body=%q", partial.Code, partial.Body.String())
	}
	if missing := request(http.MethodGet, "/api/android/v1/document-assets/vault/missing/Manual.pdf", token, ""); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown item status=%d", missing.Code)
	}
	if err := devices.revoke(m.readingAccount(nil), deviceID); err != nil {
		t.Fatal(err)
	}
	if denied := request(http.MethodGet, path, token, ""); denied.Code != http.StatusUnauthorized {
		t.Fatalf("revoked device status=%d", denied.Code)
	}
}
