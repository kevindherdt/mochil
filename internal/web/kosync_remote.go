package web

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type koSyncRemote struct {
	baseURL  string
	username string
	key      string
	client   *http.Client
}

func newKoSyncRemote(baseURL, username, password string) *koSyncRemote {
	sum := md5.Sum([]byte(password))
	return &koSyncRemote{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		key:      hex.EncodeToString(sum[:]),
		client: &http.Client{
			Timeout: 8 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return errors.New("KoSync redirects are not allowed")
			},
		},
	}
}

func (remote *koSyncRemote) progress(ctx context.Context, document string) (koSyncProgress, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, remote.baseURL+"/syncs/progress/"+document, nil)
	if err != nil {
		return koSyncProgress{}, false, err
	}
	request.Header.Set("Accept", koSyncMediaType)
	request.Header.Set("x-auth-user", remote.username)
	request.Header.Set("x-auth-key", remote.key)
	response, err := remote.client.Do(request)
	if err != nil {
		return koSyncProgress{}, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return koSyncProgress{}, false, nil
	}
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return koSyncProgress{}, false, fmt.Errorf("KoSync server returned HTTP %d", response.StatusCode)
	}
	var progress koSyncProgress
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&progress); err != nil {
		return koSyncProgress{}, false, err
	}
	if progress.Document == "" && progress.Progress == "" {
		return koSyncProgress{}, false, nil
	}
	return progress, true, nil
}
