package secrets

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"filippo.io/age"
)

// Store manages age-encrypted key/value secrets persisted to disk.
type Store struct {
	identityPath string
	secretsPath  string
	identity     *age.X25519Identity
	data         map[string]string
}

// Open loads the secrets store from dir, generating a new identity and empty
// secrets file if they don't already exist. dir must already exist.
func Open(dir string) (*Store, error) {
	identityPath := filepath.Join(dir, "identity.age")
	secretsPath := filepath.Join(dir, "secrets.age")

	identity, err := loadOrCreateIdentity(identityPath)
	if err != nil {
		return nil, fmt.Errorf("secrets: identity: %w", err)
	}

	s := &Store{
		identityPath: identityPath,
		secretsPath:  secretsPath,
		identity:     identity,
		data:         make(map[string]string),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Get returns the value for key, or ("", false) if not set.
func (s *Store) Get(key string) (string, bool) {
	v, ok := s.data[key]
	return v, ok
}

// Set stores or updates key=value and persists to disk.
func (s *Store) Set(key, value string) error {
	s.data[key] = value
	return s.save()
}

// Delete removes key and persists to disk. Returns an error if the key does
// not exist.
func (s *Store) Delete(key string) error {
	if _, ok := s.data[key]; !ok {
		return fmt.Errorf("secrets: key %q not found", key)
	}
	delete(s.data, key)
	return s.save()
}

// Keys returns all stored key names in sorted order.
func (s *Store) Keys() []string {
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func loadOrCreateIdentity(path string) (*age.X25519Identity, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return createIdentity(path)
	}
	if err != nil {
		return nil, err
	}
	identities, err := age.ParseIdentities(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	if len(identities) == 0 {
		return nil, fmt.Errorf("no identity found in %s", path)
	}
	id, ok := identities[0].(*age.X25519Identity)
	if !ok {
		return nil, fmt.Errorf("unexpected identity type in %s", path)
	}
	return id, nil
}

func createIdentity(path string) (*age.X25519Identity, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(id.String()+"\n"), 0o600); err != nil {
		return nil, err
	}
	return id, nil
}

func (s *Store) load() error {
	f, err := os.Open(s.secretsPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("secrets: open: %w", err)
	}
	defer func() { _ = f.Close() }()

	r, err := age.Decrypt(f, s.identity)
	if err != nil {
		return fmt.Errorf("secrets: decrypt: %w", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("secrets: read: %w", err)
	}
	return json.Unmarshal(data, &s.data)
}

func (s *Store) save() error {
	data, err := json.Marshal(s.data)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.secretsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("secrets: create: %w", err)
	}
	defer func() { _ = f.Close() }()

	w, err := age.Encrypt(f, s.identity.Recipient())
	if err != nil {
		return fmt.Errorf("secrets: encrypt: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("secrets: write: %w", err)
	}
	return w.Close()
}
