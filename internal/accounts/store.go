// Package accounts owns local authentication identities independently from
// browser sessions and account-scoped media state.
package accounts

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
	"mochil/internal/statefile"
)

const (
	maximumStateBytes    int64 = 4 << 20
	maximumAccounts            = 256
	maximumPasswordBytes       = 72 // bcrypt reads no more than this; a longer one is refused up front, not with a server error
	RoleAdministrator          = "administrator"
	RoleUser                   = "user"
)

var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrInvalid               = errors.New("invalid account")
	ErrConflict              = errors.New("account already exists")
	ErrLastAdministrator     = errors.New("at least one enabled administrator is required")
	ErrAccountMustBeDisabled = errors.New("account must be disabled")
)

var dummyPasswordHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

type Identity struct {
	ID       string
	Username string
	Role     string
}

type Summary struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Disabled bool   `json:"disabled"`
}

// passwordChecks bounds how many password hashes are compared at the same moment. A burst of sign-in attempts then queues
// instead of running dozens of expensive comparisons in parallel, and the rest of the server keeps its CPU.
var passwordChecks = make(chan struct{}, max(2, runtime.NumCPU()))

func comparePassword(hash, password []byte) error {
	passwordChecks <- struct{}{}
	defer func() { <-passwordChecks }()
	return compareHash(hash, password)
}

var compareHash = bcrypt.CompareHashAndPassword

func (i Identity) Administrator() bool { return i.Role == RoleAdministrator }

type account struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Normalized   string `json:"normalized_username"`
	PasswordHash string `json:"password_hash"`
	Role         string `json:"role"`
	Disabled     bool   `json:"disabled,omitempty"`
}

type state struct {
	Version         int       `json:"version"`
	RecoveryEnabled bool      `json:"environment_recovery_enabled"`
	Accounts        []account `json:"accounts"`
}

type Store struct {
	path       string
	mu         sync.RWMutex
	data       state
	byUsername map[string]int
}

// Open bootstraps exactly one administrator only when no account state exists.
// Its immutable ID deliberately equals the legacy username so existing
// account-keyed progress remains in place during migration.
func Open(path, bootstrapUsername, bootstrapPassword string) (*Store, error) {
	store := &Store{path: path, data: state{Version: 2, RecoveryEnabled: true, Accounts: []account{}}, byUsername: map[string]int{}}
	if path != "" {
		if err := statefile.DecodeJSON(path, maximumStateBytes, &store.data); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("decode accounts: %w", err)
			}
			if bootstrapUsername == "" || bootstrapPassword == "" {
				return nil, errors.New("initial administrator credentials are required")
			}
			// The bootstrap password is not exempt from the policy the rest of the
			// store enforces; >72 bytes would otherwise abort startup inside bcrypt.
			if !validUsername(bootstrapUsername) {
				return nil, errors.New("the initial administrator username is not valid")
			}
			if !validNewPassword(bootstrapPassword) {
				return nil, fmt.Errorf("the initial administrator password must be between %d and %d bytes", 10, maximumPasswordBytes)
			}
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(bootstrapPassword), bcrypt.DefaultCost)
			if hashErr != nil {
				return nil, fmt.Errorf("hash initial administrator password: %w", hashErr)
			}
			store.data.Accounts = append(store.data.Accounts, account{ID: bootstrapUsername, Username: bootstrapUsername, Normalized: normalizeUsername(bootstrapUsername), PasswordHash: string(hash), Role: RoleAdministrator})
			if err := store.validateAndIndex(); err != nil {
				return nil, err
			}
			if err := store.persistLocked(); err != nil {
				return nil, fmt.Errorf("save initial administrator: %w", err)
			}
			return store, nil
		}
		if store.data.Version == 1 {
			store.data.Version = 2
			store.data.RecoveryEnabled = true
			if err := store.validateAndIndex(); err != nil {
				return nil, err
			}
			if err := store.persistLocked(); err != nil {
				return nil, fmt.Errorf("migrate account state: %w", err)
			}
			return store, nil
		}
	}
	if err := store.validateAndIndex(); err != nil {
		return nil, err
	}
	return store, nil
}

// OpenFirstRun opens the account store without seeding anything, so an install that has never had
// an administrator starts empty and creates the first one through the first-run page. Unlike Open,
// a missing file is not an error: nothing is written until the first account is created.
func OpenFirstRun(path string) (*Store, error) {
	store := &Store{path: path, data: state{Version: 2, RecoveryEnabled: true, Accounts: []account{}}, byUsername: map[string]int{}}
	if path != "" {
		if err := statefile.DecodeJSON(path, maximumStateBytes, &store.data); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("decode accounts: %w", err)
			}
			return store, nil
		}
		if store.data.Version == 1 {
			store.data.Version = 2
			store.data.RecoveryEnabled = true
			if err := store.validateAndIndex(); err != nil {
				return nil, err
			}
			if err := store.persistLocked(); err != nil {
				return nil, fmt.Errorf("migrate account state: %w", err)
			}
			return store, nil
		}
	}
	if err := store.validateAndIndex(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Authenticate(username, password string) (Identity, error) {
	normalized := normalizeUsername(username)
	s.mu.RLock()
	index, ok := s.byUsername[normalized]
	if !ok {
		s.mu.RUnlock()
		_ = comparePassword(dummyPasswordHash, []byte(password))
		return Identity{}, ErrInvalidCredentials
	}
	value := s.data.Accounts[index]
	s.mu.RUnlock()
	// Compare even when the account is disabled: skipping bcrypt for it answered in
	// microseconds and told an unauthenticated caller which accounts exist and are off.
	bad := comparePassword([]byte(value.PasswordHash), []byte(password)) != nil
	if value.Disabled || bad {
		return Identity{}, ErrInvalidCredentials
	}
	return Identity{ID: value.ID, Username: value.Username, Role: value.Role}, nil
}

func (s *Store) InitialAdministrator() (Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, value := range s.data.Accounts {
		if value.Role == RoleAdministrator && !value.Disabled {
			return Identity{ID: value.ID, Username: value.Username, Role: value.Role}, true
		}
	}
	return Identity{}, false
}

func (s *Store) IdentityByID(id string) (Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, value := range s.data.Accounts {
		if value.ID == id && !value.Disabled {
			return Identity{ID: value.ID, Username: value.Username, Role: value.Role}, true
		}
	}
	return Identity{}, false
}

func (s *Store) IdentityByUsername(username string) (Identity, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	index, ok := s.byUsername[normalizeUsername(username)]
	if !ok || s.data.Accounts[index].Disabled {
		return Identity{}, false
	}
	value := s.data.Accounts[index]
	return Identity{ID: value.ID, Username: value.Username, Role: value.Role}, true
}

func (s *Store) List() []Summary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Summary, 0, len(s.data.Accounts))
	for _, value := range s.data.Accounts {
		items = append(items, Summary{ID: value.ID, Username: value.Username, Role: value.Role, Disabled: value.Disabled})
	}
	return items
}

func (s *Store) RecoveryEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.RecoveryEnabled
}

func (s *Store) ConfirmRecovery(id, password string) error {
	// bcrypt takes tens of milliseconds; running it under the store's write lock
	// would stall every authenticated request for that long.
	s.mu.RLock()
	index := s.indexByIDLocked(id)
	invalid := index < 0 || s.data.Accounts[index].Disabled || s.data.Accounts[index].Role != RoleAdministrator
	hash := ""
	if !invalid {
		hash = s.data.Accounts[index].PasswordHash
	}
	s.mu.RUnlock()
	if invalid || comparePassword([]byte(hash), []byte(password)) != nil {
		return ErrInvalidCredentials
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.data.RecoveryEnabled {
		return nil
	}
	s.data.RecoveryEnabled = false
	if err := s.persistLocked(); err != nil {
		s.data.RecoveryEnabled = true
		return err
	}
	return nil
}

func (s *Store) Create(username, password, role string) (Identity, error) {
	if !validUsername(username) || !validNewPassword(password) || (role != RoleUser && role != RoleAdministrator) {
		return Identity{}, ErrInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Identity{}, err
	}
	id, err := randomID()
	if err != nil {
		return Identity{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data.Accounts) >= maximumAccounts {
		return Identity{}, ErrInvalid
	}
	normalized := normalizeUsername(username)
	if _, exists := s.byUsername[normalized]; exists {
		return Identity{}, ErrConflict
	}
	previous := append([]account(nil), s.data.Accounts...)
	s.data.Accounts = append(s.data.Accounts, account{ID: id, Username: username, Normalized: normalized, PasswordHash: string(hash), Role: role})
	s.byUsername[normalized] = len(s.data.Accounts) - 1
	if err := s.persistLocked(); err != nil {
		s.data.Accounts = previous
		_ = s.validateAndIndex()
		return Identity{}, err
	}
	return Identity{ID: id, Username: username, Role: role}, nil
}

func (s *Store) SetDisabled(id string, disabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexByIDLocked(id)
	if index < 0 {
		return ErrInvalid
	}
	if disabled && s.data.Accounts[index].Role == RoleAdministrator && s.enabledAdministratorCountLocked() == 1 {
		return ErrLastAdministrator
	}
	previous := s.data.Accounts[index].Disabled
	s.data.Accounts[index].Disabled = disabled
	if err := s.persistLocked(); err != nil {
		s.data.Accounts[index].Disabled = previous
		return err
	}
	return nil
}

func (s *Store) ResetPassword(id, password string) error {
	if !validNewPassword(password) {
		return ErrInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexByIDLocked(id)
	if index < 0 {
		return ErrInvalid
	}
	previous := s.data.Accounts[index].PasswordHash
	s.data.Accounts[index].PasswordHash = string(hash)
	if err := s.persistLocked(); err != nil {
		s.data.Accounts[index].PasswordHash = previous
		return err
	}
	return nil
}

func (s *Store) DeleteDisabled(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexByIDLocked(id)
	if index < 0 {
		return ErrInvalid
	}
	if !s.data.Accounts[index].Disabled {
		return ErrAccountMustBeDisabled
	}
	if len(s.data.Accounts) == 1 {
		return ErrLastAdministrator
	}
	previous := append([]account(nil), s.data.Accounts...)
	s.data.Accounts = append(s.data.Accounts[:index], s.data.Accounts[index+1:]...)
	if err := s.validateAndIndex(); err != nil {
		s.data.Accounts = previous
		_ = s.validateAndIndex()
		return err
	}
	if err := s.persistLocked(); err != nil {
		s.data.Accounts = previous
		_ = s.validateAndIndex()
		return err
	}
	return nil
}

func (s *Store) indexByIDLocked(id string) int {
	for index := range s.data.Accounts {
		if s.data.Accounts[index].ID == id {
			return index
		}
	}
	return -1
}
func (s *Store) enabledAdministratorCountLocked() int {
	count := 0
	for _, value := range s.data.Accounts {
		if value.Role == RoleAdministrator && !value.Disabled {
			count++
		}
	}
	return count
}

func (s *Store) validateAndIndex() error {
	if s.data.Version != 2 {
		return fmt.Errorf("unsupported account-state version %d", s.data.Version)
	}
	if len(s.data.Accounts) == 0 || len(s.data.Accounts) > maximumAccounts {
		return errors.New("account state has an invalid account count")
	}
	s.byUsername = make(map[string]int, len(s.data.Accounts))
	ids, administrators := map[string]bool{}, 0
	for index, value := range s.data.Accounts {
		if !validID(value.ID) || !validUsername(value.Username) || value.Normalized != normalizeUsername(value.Username) || value.PasswordHash == "" {
			return errors.New("account state contains an invalid account")
		}
		if value.Role != RoleAdministrator && value.Role != RoleUser {
			return errors.New("account state contains an invalid role")
		}
		_, duplicateUsername := s.byUsername[value.Normalized]
		if ids[value.ID] || duplicateUsername {
			return errors.New("account state contains a duplicate identity")
		}
		if _, err := bcrypt.Cost([]byte(value.PasswordHash)); err != nil {
			return errors.New("account state contains an invalid password hash")
		}
		ids[value.ID], s.byUsername[value.Normalized] = true, index
		if value.Role == RoleAdministrator && !value.Disabled {
			administrators++
		}
	}
	if administrators == 0 {
		return errors.New("account state has no enabled administrator")
	}
	return nil
}

func (s *Store) persistLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.path), ".accounts-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err = temporary.Chmod(0o600); err == nil {
		err = statefile.EncodeJSON(temporary, maximumStateBytes, s.data, false)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, s.path)
}

func normalizeUsername(value string) string {
	return cases.Fold().String(norm.NFKC.String(strings.TrimSpace(value)))
}

func validUsername(value string) bool {
	return value == strings.TrimSpace(value) && utf8.ValidString(value) && len(value) >= 1 && len(value) <= 128 && normalizeUsername(value) != ""
}
func validID(value string) bool {
	return len(value) >= 1 && len(value) <= 128 && utf8.ValidString(value) && !strings.ContainsAny(value, "\x00\r\n")
}
func validNewPassword(value string) bool {
	return len(value) >= 10 && len(value) <= maximumPasswordBytes && utf8.ValidString(value)
}

func randomID() (string, error) {
	data := make([]byte, 18)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return "acct_" + base64.RawURLEncoding.EncodeToString(data), nil
}
