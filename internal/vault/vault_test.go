package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withTempVault overrides VaultPathFunc to use a temporary directory.
// Returns a cleanup function to restore the original.
func withTempVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.dat")
	orig := VaultPathFunc
	VaultPathFunc = func() string { return path }
	t.Cleanup(func() { VaultPathFunc = orig })
	return path
}

func sampleEntry(name string) *Entry {
	return &Entry{
		ID:        "testid-" + name,
		Name:      name,
		Username:  "user@example.com",
		Password:  "s3cr3t",
		URL:       "https://example.com",
		Notes:     "some notes",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestVaultPath_ContainsExpectedSuffix(t *testing.T) {
	path := VaultPath()
	if !strings.HasSuffix(path, filepath.Join(".0pass", "vault.dat")) {
		t.Fatalf("unexpected vault path: %s", path)
	}
}

func TestInit_CreatesVaultFile(t *testing.T) {
	vaultPath := withTempVault(t)
	pw := []byte("masterpassword")

	if err := Init(pw); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < len(magic)+1+saltLen+nonceLen {
		t.Fatal("vault file too short")
	}
	// Check magic header
	if string(data[:4]) != "0PAS" {
		t.Fatalf("bad magic header: %q", data[:4])
	}
	// Check version
	if data[4] != version {
		t.Fatalf("bad version: got %d, want %d", data[4], version)
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	withTempVault(t)
	pw := []byte("pw")

	if err := Init(pw); err != nil {
		t.Fatal(err)
	}
	err := Init(pw)
	if err == nil {
		t.Fatal("expected error when vault already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_NotFound(t *testing.T) {
	withTempVault(t)
	_, err := Load([]byte("pw"))
	if err == nil {
		t.Fatal("expected error for missing vault")
	}
	if !strings.Contains(err.Error(), "vault not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_TooShort(t *testing.T) {
	vaultPath := withTempVault(t)
	_ = os.WriteFile(vaultPath, []byte("short"), 0600)

	_, err := Load([]byte("pw"))
	if err == nil {
		t.Fatal("expected error for short file")
	}
	if !strings.Contains(err.Error(), "corrupted") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_BadMagic(t *testing.T) {
	vaultPath := withTempVault(t)
	data := make([]byte, len(magic)+1+saltLen+nonceLen+20)
	copy(data, []byte("XXXX")) // bad magic
	_ = os.WriteFile(vaultPath, data, 0600)

	_, err := Load([]byte("pw"))
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
	if !strings.Contains(err.Error(), "corrupted") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_BadVersion(t *testing.T) {
	vaultPath := withTempVault(t)
	data := make([]byte, len(magic)+1+saltLen+nonceLen+20)
	copy(data, magic)
	data[4] = 0xFF // unsupported version
	_ = os.WriteFile(vaultPath, data, 0600)

	_, err := Load([]byte("pw"))
	if err == nil {
		t.Fatal("expected error for bad version")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_WrongPassword(t *testing.T) {
	withTempVault(t)
	pw := []byte("correctpassword")

	if err := Init(pw); err != nil {
		t.Fatal(err)
	}
	_, err := Load([]byte("wrongpassword"))
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if !strings.Contains(err.Error(), "wrong password") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSave_Roundtrip(t *testing.T) {
	withTempVault(t)
	pw := []byte("testpassword")

	if err := Init(pw); err != nil {
		t.Fatal(err)
	}

	v, err := Load(pw)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Entries) != 0 {
		t.Fatalf("new vault should be empty, got %d entries", len(v.Entries))
	}

	entry := sampleEntry("GitHub")
	v.Entries = append(v.Entries, entry)

	if err := v.Save(pw); err != nil {
		t.Fatal(err)
	}

	v2, err := Load(pw)
	if err != nil {
		t.Fatal(err)
	}
	if len(v2.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v2.Entries))
	}
	if v2.Entries[0].Name != "GitHub" {
		t.Fatalf("wrong entry name: %s", v2.Entries[0].Name)
	}
	if v2.Entries[0].Password != "s3cr3t" {
		t.Fatal("password mismatch after roundtrip")
	}
}

func TestLoadSave_MultipleEntries(t *testing.T) {
	withTempVault(t)
	pw := []byte("pw")

	if err := Init(pw); err != nil {
		t.Fatal(err)
	}

	v, err := Load(pw)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		v.Entries = append(v.Entries, sampleEntry(name))
	}
	if err := v.Save(pw); err != nil {
		t.Fatal(err)
	}

	v2, err := Load(pw)
	if err != nil {
		t.Fatal(err)
	}
	if len(v2.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(v2.Entries))
	}
}

func TestSave_GeneratesNewSaltEachTime(t *testing.T) {
	vaultPath := withTempVault(t)
	pw := []byte("pw")

	if err := Init(pw); err != nil {
		t.Fatal(err)
	}

	data1, _ := os.ReadFile(vaultPath)
	salt1 := data1[5 : 5+saltLen]

	v, _ := Load(pw)
	_ = v.Save(pw)

	data2, _ := os.ReadFile(vaultPath)
	salt2 := data2[5 : 5+saltLen]

	for i := range salt1 {
		if salt1[i] != salt2[i] {
			return // they differ, as expected
		}
	}
	t.Fatal("expected different salts on each save")
}

func TestSave_NoTempFileLeft(t *testing.T) {
	dir := t.TempDir()
	orig := VaultPathFunc
	vaultPath := filepath.Join(dir, "vault.dat")
	VaultPathFunc = func() string { return vaultPath }
	t.Cleanup(func() { VaultPathFunc = orig })

	pw := []byte("pw")
	_ = Init(pw)
	v, _ := Load(pw)
	_ = v.Save(pw)

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestFindEntries_EmptyQuery(t *testing.T) {
	withTempVault(t)
	pw := []byte("pw")
	_ = Init(pw)
	v, _ := Load(pw)
	v.Entries = []*Entry{sampleEntry("Alpha"), sampleEntry("Beta")}

	results := v.FindEntries("")
	if len(results) != 2 {
		t.Fatalf("empty query should return all, got %d", len(results))
	}
}

func TestFindEntries_Match(t *testing.T) {
	v := &Vault{Entries: []*Entry{
		sampleEntry("GitHub"),
		sampleEntry("Gmail"),
		sampleEntry("Notion"),
	}}

	results := v.FindEntries("github")
	if len(results) == 0 {
		t.Fatal("expected at least one match for 'github'")
	}
	if results[0].Name != "GitHub" {
		t.Fatalf("expected GitHub as top match, got %s", results[0].Name)
	}
}

func TestFindEntries_NoMatch(t *testing.T) {
	v := &Vault{Entries: []*Entry{sampleEntry("GitHub")}}
	results := v.FindEntries("zzznomatch")
	if len(results) != 0 {
		t.Fatalf("expected no matches, got %d", len(results))
	}
}

func TestFindEntries_MatchesURL(t *testing.T) {
	e := sampleEntry("MyService")
	e.URL = "https://myservice.example.org"
	v := &Vault{Entries: []*Entry{e}}

	results := v.FindEntries("myservice")
	if len(results) == 0 {
		t.Fatal("expected match on URL")
	}
}

func TestFindEntries_MatchesUsername(t *testing.T) {
	e := sampleEntry("SomeService")
	e.Username = "alice@wonderland.com"
	v := &Vault{Entries: []*Entry{e}}

	results := v.FindEntries("alice")
	if len(results) == 0 {
		t.Fatal("expected match on username")
	}
}

func TestInit_BadDir(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file where the vault directory should be,
	// so MkdirAll fails trying to create a directory over a file.
	blockingFile := filepath.Join(dir, "notadir")
	if err := os.WriteFile(blockingFile, []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}

	orig := VaultPathFunc
	VaultPathFunc = func() string { return filepath.Join(blockingFile, "vault.dat") }
	t.Cleanup(func() { VaultPathFunc = orig })

	err := Init([]byte("pw"))
	if err == nil {
		t.Fatal("expected error when vault directory cannot be created")
	}
}

func TestSave_WriteError(t *testing.T) {
	dir := t.TempDir()
	orig := VaultPathFunc
	vaultPath := filepath.Join(dir, "vault.dat")
	VaultPathFunc = func() string { return vaultPath }
	t.Cleanup(func() { VaultPathFunc = orig })

	// Init the vault first
	pw := []byte("pw")
	if err := Init(pw); err != nil {
		t.Fatal(err)
	}
	v, err := Load(pw)
	if err != nil {
		t.Fatal(err)
	}

	// Make the directory read-only so WriteFile fails
	if err := os.Chmod(dir, 0500); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0700) }() //nolint:gosec

	err = v.Save(pw)
	if err == nil {
		t.Fatal("expected error when vault directory is read-only")
	}
}

func TestLoad_ReadError(t *testing.T) {
	dir := t.TempDir()
	orig := VaultPathFunc
	vaultPath := filepath.Join(dir, "vault.dat")
	VaultPathFunc = func() string { return vaultPath }
	t.Cleanup(func() { VaultPathFunc = orig })

	// Create an unreadable vault file
	if err := os.WriteFile(vaultPath, []byte("data"), 0200); err != nil { // write-only
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(vaultPath, 0600) }()

	_, err := Load([]byte("pw"))
	if err == nil {
		t.Fatal("expected error for unreadable vault file")
	}
	if strings.Contains(err.Error(), "vault not found") {
		t.Fatalf("wrong error type, got: %v", err)
	}
}

func TestVaultPath_FallbackWhenHomedirFails(t *testing.T) {
	orig := userHomeDir
	userHomeDir = func() (string, error) { return "", fmt.Errorf("no home dir") }
	t.Cleanup(func() { userHomeDir = orig })

	path := VaultPathFunc()
	if path != ".0pass/vault.dat" {
		t.Errorf("expected fallback path, got %s", path)
	}
}

func TestSave_RandError(t *testing.T) {
	withTempVault(t)
	pw := []byte("pw")
	if err := Init(pw); err != nil {
		t.Fatal(err)
	}
	v, err := Load(pw)
	if err != nil {
		t.Fatal(err)
	}

	// Make randReader fail so newSalt() in save() returns an error.
	origRand := randReader
	randReader = &errorReader{}
	t.Cleanup(func() { randReader = origRand })

	err = v.Save(pw)
	if err == nil {
		t.Fatal("expected error when randReader fails during save")
	}
}

func TestSave_RenameError(t *testing.T) {
	withTempVault(t)
	pw := []byte("pw")
	if err := Init(pw); err != nil {
		t.Fatal(err)
	}
	v, err := Load(pw)
	if err != nil {
		t.Fatal(err)
	}

	origRename := osRename
	osRename = func(oldpath, newpath string) error {
		return fmt.Errorf("rename failed")
	}
	t.Cleanup(func() { osRename = origRename })

	err = v.Save(pw)
	if err == nil {
		t.Fatal("expected error when rename fails")
	}
	if !strings.Contains(err.Error(), "saving vault") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoad_NilEntries(t *testing.T) {
	withTempVault(t)
	pw := []byte("pw")

	// Save a vault with nil Entries to produce {"entries":null} in the JSON.
	v := &Vault{} // Entries is nil
	if err := save(v, pw, VaultPath()); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(pw)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Entries == nil {
		t.Fatal("expected non-nil entries slice after loading vault with null entries")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	vaultPath := withTempVault(t)
	pw := []byte("pw")

	// Construct a valid-format vault file whose plaintext is not valid JSON.
	salt, err := newSalt()
	if err != nil {
		t.Fatal(err)
	}
	key := deriveKey(pw, salt)
	nonce, ciphertext, err := encrypt(key, []byte("this is not valid json {{{{"))
	ZeroBytes(key)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.Write(magic)
	buf.WriteByte(version)
	buf.Write(salt)
	buf.Write(nonce)
	buf.Write(ciphertext)

	if err := os.WriteFile(vaultPath, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}

	_, err = Load(pw)
	if err == nil {
		t.Fatal("expected error for invalid JSON plaintext")
	}
	if !strings.Contains(err.Error(), "wrong password or corrupted vault") {
		t.Errorf("unexpected error: %v", err)
	}
}
