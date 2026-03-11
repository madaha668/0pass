package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sahilm/fuzzy"
)

var magic = []byte("0PAS")

const version byte = 0x01

// userHomeDir is injectable for tests.
var userHomeDir = os.UserHomeDir

// osRename is injectable for tests.
var osRename = os.Rename

// VaultPathFunc returns the path to the vault file.
// It can be overridden in tests.
var VaultPathFunc = func() string {
	home, err := userHomeDir()
	if err != nil {
		return ".0pass/vault.dat"
	}
	return filepath.Join(home, ".0pass", "vault.dat")
}

// VaultPath returns the current vault file path.
func VaultPath() string { return VaultPathFunc() }

// Vault holds all password entries.
type Vault struct {
	Entries []*Entry `json:"entries"`
}

// Init creates a new empty vault at the default path.
// Returns an error if the vault already exists.
func Init(password []byte) error {
	path := VaultPath()
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("vault already exists at %s", path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating vault directory: %w", err)
	}

	v := &Vault{Entries: []*Entry{}}
	return save(v, password, path)
}

// Load decrypts and loads the vault from the default path.
func Load(password []byte) (*Vault, error) {
	return load(password, VaultPath())
}

// Save encrypts and saves the vault to the default path.
func (v *Vault) Save(password []byte) error {
	return save(v, password, VaultPath())
}

// save writes the vault to path+".tmp" then renames for atomicity.
// A new salt is generated on every save so the encryption key is refreshed.
func save(v *Vault, password []byte, path string) error {
	plaintext, _ := json.Marshal(v)

	salt, err := newSalt()
	if err != nil {
		return err
	}

	key := deriveKey(password, salt)
	defer ZeroBytes(key)

	nonce, ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.Write(magic)
	buf.WriteByte(version)
	buf.Write(salt)
	buf.Write(nonce)
	buf.Write(ciphertext)

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("writing vault: %w", err)
	}

	if err := osRename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("saving vault: %w", err)
	}

	return nil
}

// load reads and decrypts the vault from path.
func load(password []byte, path string) (*Vault, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("vault not found; run '0pass init' first")
		}
		return nil, fmt.Errorf("reading vault: %w", err)
	}

	// Parse header: 4 magic + 1 version + 32 salt + 12 nonce = 49 bytes minimum
	minLen := len(magic) + 1 + saltLen + nonceLen
	if len(data) < minLen {
		return nil, fmt.Errorf("wrong password or corrupted vault")
	}

	offset := 0

	// Check magic
	if !bytes.Equal(data[offset:offset+len(magic)], magic) {
		return nil, fmt.Errorf("wrong password or corrupted vault")
	}
	offset += len(magic)

	// Check version
	if data[offset] != version {
		return nil, fmt.Errorf("unsupported vault version %d", data[offset])
	}
	offset++

	// Read salt
	salt := data[offset : offset+saltLen]
	offset += saltLen

	// Read nonce
	nonce := data[offset : offset+nonceLen]
	offset += nonceLen

	// Remaining is ciphertext
	ciphertext := data[offset:]

	key := deriveKey(password, salt)
	defer ZeroBytes(key)

	plaintext, err := decrypt(key, nonce, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("wrong password or corrupted vault")
	}

	var v Vault
	if err := json.Unmarshal(plaintext, &v); err != nil {
		return nil, fmt.Errorf("wrong password or corrupted vault")
	}

	if v.Entries == nil {
		v.Entries = []*Entry{}
	}

	return &v, nil
}

// entrySource implements fuzzy.Source for vault entries.
type entrySource struct{ entries []*Entry }

func (s entrySource) String(i int) string {
	e := s.entries[i]
	return e.Name + " " + e.URL + " " + e.Username
}

func (s entrySource) Len() int { return len(s.entries) }

// FindEntries performs a fuzzy search over entries by Name, URL, and Username.
// If query is empty, all entries are returned.
func (v *Vault) FindEntries(query string) []*Entry {
	if query == "" {
		return v.Entries
	}

	src := entrySource{entries: v.Entries}
	matches := fuzzy.FindFrom(query, src)

	result := make([]*Entry, 0, len(matches))
	for _, m := range matches {
		result = append(result, v.Entries[m.Index])
	}
	return result
}
