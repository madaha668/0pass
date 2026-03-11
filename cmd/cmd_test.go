package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/madaha668/0pass/internal/fetch"
	"github.com/madaha668/0pass/internal/vault"
)

// testEnv sets up an isolated test environment with a temp vault and
// injectable I/O. All overrides are restored via t.Cleanup.
type testEnv struct {
	outBuf *bytes.Buffer
	errBuf *bytes.Buffer
	pw     []byte
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	vaultFile := filepath.Join(dir, "vault.dat")

	origVaultPath := vault.VaultPathFunc
	origStdin := stdin
	origStdout := stdout
	origStderr := stderr
	origPasswordReader := passwordReader
	origClipboardWriter := clipboardWriter
	origPageInfoFetcher := pageInfoFetcher

	env := &testEnv{
		outBuf: &bytes.Buffer{},
		errBuf: &bytes.Buffer{},
		pw:     []byte("masterpassword"),
	}

	vault.VaultPathFunc = func() string { return vaultFile }
	stdout = env.outBuf
	stderr = env.errBuf

	// silence cobra's own error/usage output
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	t.Cleanup(func() {
		vault.VaultPathFunc = origVaultPath
		stdin = origStdin
		stdout = origStdout
		stderr = origStderr
		passwordReader = origPasswordReader
		clipboardWriter = origClipboardWriter
		pageInfoFetcher = origPageInfoFetcher
		rootCmd.SilenceErrors = false
		rootCmd.SilenceUsage = false
	})

	return env
}

// setPasswords sets passwordReader to return passwords in sequence.
// Each call returns a fresh copy so that ZeroBytes in commands doesn't
// corrupt the original slices (e.g. env.pw).
func (e *testEnv) setPasswords(pws ...[]byte) {
	copies := make([][]byte, len(pws))
	for i, pw := range pws {
		c := make([]byte, len(pw))
		copy(c, pw)
		copies[i] = c
	}
	i := 0
	passwordReader = func(prompt string) ([]byte, error) {
		if i >= len(copies) {
			return nil, fmt.Errorf("no more passwords configured")
		}
		pw := copies[i]
		i++
		return pw, nil
	}
}

// setStdinLines sets stdin to return lines in sequence.
func (e *testEnv) setStdinLines(lines ...string) {
	stdin = strings.NewReader(strings.Join(lines, "\n") + "\n")
}

// pwCopy returns a fresh copy of env.pw safe to pass to functions that zero it.
func (e *testEnv) pwCopy() []byte {
	c := make([]byte, len(e.pw))
	copy(c, e.pw)
	return c
}

// initVault initializes the test vault with env.pw.
func (e *testEnv) initVault(t *testing.T) {
	t.Helper()
	if err := vault.Init(e.pwCopy()); err != nil {
		t.Fatal(err)
	}
}

// addEntry adds a test entry directly to the vault.
func (e *testEnv) addEntry(t *testing.T, name, username, password, url, notes string) *vault.Entry {
	t.Helper()
	v, err := vault.Load(e.pwCopy())
	if err != nil {
		t.Fatal(err)
	}
	entry := &vault.Entry{
		ID:        "id-" + name,
		Name:      name,
		Username:  username,
		Password:  password,
		URL:       url,
		Notes:     notes,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	v.Entries = append(v.Entries, entry)
	if err := v.Save(e.pwCopy()); err != nil {
		t.Fatal(err)
	}
	return entry
}

// run executes a command and returns its error.
func (e *testEnv) run(args ...string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// --- init command ---

func TestInitCommand_Success(t *testing.T) {
	env := newTestEnv(t)
	env.setPasswords(env.pw, env.pw)

	if err := env.run("init"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "Vault initialized") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}
}

func TestInitCommand_PasswordMismatch(t *testing.T) {
	env := newTestEnv(t)
	env.setPasswords([]byte("first"), []byte("second"))

	err := env.run("init")
	if err == nil {
		t.Fatal("expected error for mismatched passwords")
	}
	if !strings.Contains(err.Error(), "do not match") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitCommand_AlreadyExists(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords(env.pw, env.pw)

	err := env.run("init")
	if err == nil {
		t.Fatal("expected error when vault already exists")
	}
}

// --- add command ---

func TestAddCommand_ManualPassword(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	pageInfoFetcher = func(url string) (*fetch.PageInfo, error) {
		return nil, fmt.Errorf("no fetch")
	}
	env.setPasswords(env.pw, []byte("mypassword"))
	env.setStdinLines("GitHub", "alice", "https://github.com", "code hosting")

	if err := env.run("add"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "Entry added") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}

	v, _ := vault.Load(env.pwCopy())
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
	if v.Entries[0].Name != "GitHub" {
		t.Errorf("wrong name: %s", v.Entries[0].Name)
	}
	if v.Entries[0].Password != "mypassword" {
		t.Error("password not saved correctly")
	}
}

func TestAddCommand_GeneratedPassword(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	pageInfoFetcher = func(url string) (*fetch.PageInfo, error) {
		return nil, fmt.Errorf("no fetch")
	}
	// Empty password triggers generation
	env.setPasswords(env.pw, []byte(""))
	env.setStdinLines("MyApp", "bob", "https://myapp.io", "my notes")

	if err := env.run("add"); err != nil {
		t.Fatal(err)
	}
	out := env.outBuf.String()
	if !strings.Contains(out, "Generated password:") {
		t.Errorf("expected generated password message, got: %s", out)
	}

	v, _ := vault.Load(env.pwCopy())
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
	if v.Entries[0].Password == "" {
		t.Error("generated password should not be empty")
	}
}

func TestAddCommand_AcceptFetchedNotes(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	pageInfoFetcher = func(url string) (*fetch.PageInfo, error) {
		return &fetch.PageInfo{Title: "Test Site", Description: "A test"}, nil
	}
	env.setPasswords(env.pw, []byte("pw123"))
	// After fetch: answer "y" to use fetched notes
	env.setStdinLines("TestSite", "user", "https://test.io", "y")

	if err := env.run("add"); err != nil {
		t.Fatal(err)
	}

	v, _ := vault.Load(env.pwCopy())
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
	if !strings.Contains(v.Entries[0].Notes, "Test Site") {
		t.Errorf("expected fetched notes, got: %s", v.Entries[0].Notes)
	}
}

func TestAddCommand_RejectFetchedNotes(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	pageInfoFetcher = func(url string) (*fetch.PageInfo, error) {
		return &fetch.PageInfo{Title: "Test Site", Description: "A test"}, nil
	}
	env.setPasswords(env.pw, []byte("pw123"))
	// Reject fetched notes, provide manual
	env.setStdinLines("TestSite", "user", "https://test.io", "n", "manual notes")

	if err := env.run("add"); err != nil {
		t.Fatal(err)
	}

	v, _ := vault.Load(env.pwCopy())
	if v.Entries[0].Notes != "manual notes" {
		t.Errorf("expected manual notes, got: %s", v.Entries[0].Notes)
	}
}

func TestAddCommand_EmptyNameRetries(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	pageInfoFetcher = func(url string) (*fetch.PageInfo, error) {
		return nil, fmt.Errorf("no fetch")
	}
	env.setPasswords(env.pw, []byte("pw"))
	// First name is empty, second is valid
	env.setStdinLines("", "RealName", "bob", "https://x.com", "notes")

	if err := env.run("add"); err != nil {
		t.Fatal(err)
	}
	v, _ := vault.Load(env.pwCopy())
	if v.Entries[0].Name != "RealName" {
		t.Errorf("expected RealName, got: %s", v.Entries[0].Name)
	}
}

// --- get command ---

func TestGetCommand_SingleMatch_CopiesClipboard(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "secretpw", "https://github.com", "")

	var clipped string
	clipboardWriter = func(text string) error {
		clipped = text
		return nil
	}
	env.setPasswords(env.pw)

	if err := env.run("get", "github"); err != nil {
		t.Fatal(err)
	}
	if clipped != "secretpw" {
		t.Errorf("expected clipboard to contain 'secretpw', got %q", clipped)
	}
	if !strings.Contains(env.outBuf.String(), "copied to clipboard") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}
}

func TestGetCommand_NoMatch(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords(env.pw)

	if err := env.run("get", "zzznomatch"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "No entries found") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}
}

func TestGetCommand_MultipleMatches_SelectsCorrect(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw1", "https://github.com", "")
	env.addEntry(t, "GitLab", "alice", "pw2", "https://gitlab.com", "")

	var clipped string
	clipboardWriter = func(text string) error { clipped = text; return nil }
	env.setPasswords(env.pw)
	env.setStdinLines("1") // select first entry

	if err := env.run("get", "git"); err != nil {
		t.Fatal(err)
	}
	if clipped == "" {
		t.Error("expected clipboard to be set")
	}
}

func TestGetCommand_QueryFromPrompt(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "Notion", "bob", "notion-pw", "https://notion.so", "")

	var clipped string
	clipboardWriter = func(text string) error { clipped = text; return nil }
	env.setPasswords(env.pw)
	env.setStdinLines("notion") // query from prompt (no arg)

	if err := env.run("get"); err != nil {
		t.Fatal(err)
	}
	if clipped != "notion-pw" {
		t.Errorf("expected 'notion-pw', got %q", clipped)
	}
}

func TestGetCommand_InvalidSelection(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw1", "https://github.com", "")
	env.addEntry(t, "GitLab", "alice", "pw2", "https://gitlab.com", "")

	clipboardWriter = func(text string) error { return nil }
	env.setPasswords(env.pw)
	env.setStdinLines("99") // invalid selection

	err := env.run("get", "git")
	if err == nil {
		t.Fatal("expected error for invalid selection")
	}
}

// --- list command ---

func TestListCommand_Empty(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords(env.pw)

	if err := env.run("list"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "Vault is empty") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}
}

func TestListCommand_WithEntries(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw", "https://github.com", "")
	env.addEntry(t, "Gmail", "alice@gmail.com", "pw", "https://gmail.com", "")
	env.setPasswords(env.pw)

	if err := env.run("list"); err != nil {
		t.Fatal(err)
	}
	out := env.outBuf.String()
	if !strings.Contains(out, "GitHub") {
		t.Errorf("expected GitHub in output: %s", out)
	}
	if !strings.Contains(out, "Gmail") {
		t.Errorf("expected Gmail in output: %s", out)
	}
	if !strings.Contains(out, "NAME") {
		t.Errorf("expected header in output: %s", out)
	}
}

// --- edit command ---

func TestEditCommand_UpdateName(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "OldName", "alice", "pw", "https://example.com", "notes")

	env.setPasswords(env.pw, []byte(""))
	env.setStdinLines("NewName", "", "", "") // name changed, others empty (keep)

	if err := env.run("edit", "OldName"); err != nil {
		t.Fatal(err)
	}

	v, _ := vault.Load(env.pwCopy())
	if v.Entries[0].Name != "NewName" {
		t.Errorf("expected NewName, got %s", v.Entries[0].Name)
	}
	// password unchanged
	if v.Entries[0].Password != "pw" {
		t.Errorf("password should be unchanged, got %s", v.Entries[0].Password)
	}
}

func TestEditCommand_GeneratePassword(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "MyApp", "bob", "oldpw", "https://myapp.io", "notes")

	env.setPasswords(env.pw, []byte("g")) // 'g' triggers generation
	env.setStdinLines("", "", "", "")      // keep all other fields

	if err := env.run("edit", "MyApp"); err != nil {
		t.Fatal(err)
	}
	out := env.outBuf.String()
	if !strings.Contains(out, "Generated password:") {
		t.Errorf("expected generated password message: %s", out)
	}

	v, _ := vault.Load(env.pwCopy())
	if v.Entries[0].Password == "oldpw" {
		t.Error("password should have been replaced")
	}
}

func TestEditCommand_NoMatch(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords(env.pw)

	if err := env.run("edit", "zzznomatch"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "No entries found") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}
}

func TestEditCommand_MultipleMatches(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw1", "https://github.com", "")
	env.addEntry(t, "GitLab", "alice", "pw2", "https://gitlab.com", "")

	env.setPasswords(env.pw, []byte(""))
	env.setStdinLines("1", "", "", "", "") // select first, keep all fields

	if err := env.run("edit", "git"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "Entry updated") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}
}

func TestEditCommand_NoArg(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw1", "https://github.com", "")

	env.setPasswords(env.pw, []byte(""))
	env.setStdinLines("", "", "", "") // keep all fields

	if err := env.run("edit"); err != nil {
		t.Fatal(err)
	}
}

// --- delete command ---

func TestDeleteCommand_Confirm(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw", "https://github.com", "")
	env.setPasswords(env.pw)
	env.setStdinLines("y")

	if err := env.run("delete", "github"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "Deleted") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}

	v, _ := vault.Load(env.pwCopy())
	if len(v.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(v.Entries))
	}
}

func TestDeleteCommand_Cancel(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw", "https://github.com", "")
	env.setPasswords(env.pw)
	env.setStdinLines("n")

	if err := env.run("delete", "github"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "Aborted") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}

	v, _ := vault.Load(env.pwCopy())
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
}

func TestDeleteCommand_NoMatch(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords(env.pw)

	if err := env.run("delete", "zzznomatch"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "No entries found") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}
}

func TestDeleteCommand_MultipleMatches(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw1", "https://github.com", "")
	env.addEntry(t, "GitLab", "alice", "pw2", "https://gitlab.com", "")
	env.setPasswords(env.pw)
	env.setStdinLines("1", "y") // select first, confirm

	if err := env.run("delete", "git"); err != nil {
		t.Fatal(err)
	}

	v, _ := vault.Load(env.pwCopy())
	if len(v.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(v.Entries))
	}
}

// --- passwd command ---

func TestPasswdCommand_Success(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)

	newPw := []byte("newmasterpassword")
	// setPasswords makes copies, so newPw itself won't be zeroed by the command
	env.setPasswords(env.pw, newPw, newPw)

	if err := env.run("passwd"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.outBuf.String(), "Master password updated") {
		t.Errorf("unexpected output: %s", env.outBuf.String())
	}

	// verify new password works (newPw is safe: setPasswords copied it)
	_, err := vault.Load(newPw)
	if err != nil {
		t.Fatalf("vault should be loadable with new password: %v", err)
	}
	// verify old password no longer works
	_, err = vault.Load(env.pwCopy())
	if err == nil {
		t.Fatal("old password should no longer work")
	}
}

func TestPasswdCommand_WrongCurrentPassword(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords([]byte("wrongpassword"), []byte("new"), []byte("new"))

	err := env.run("passwd")
	if err == nil {
		t.Fatal("expected error for wrong current password")
	}
}

func TestPasswdCommand_NewPasswordMismatch(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords(env.pw, []byte("new1"), []byte("new2"))

	err := env.run("passwd")
	if err == nil {
		t.Fatal("expected error for mismatched new passwords")
	}
	if !strings.Contains(err.Error(), "do not match") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- error paths ---

func TestCommand_WrongMasterPassword(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords([]byte("wrongpassword"))

	err := env.run("list")
	if err == nil {
		t.Fatal("expected error for wrong master password")
	}
}

// --- Execute() ---

func TestExecute_NoError(t *testing.T) {
	// --help prints usage and returns nil (does not call os.Exit).
	// This covers the Execute() function body.
	outBuf := &bytes.Buffer{}
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(outBuf)
	SetArgs([]string{"--help"})
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	t.Cleanup(func() {
		rootCmd.SilenceErrors = false
		rootCmd.SilenceUsage = false
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})
	Execute()
}

// --- stdin error paths ---

// errReader is an io.Reader that always returns an error after any amount of data.
type errReader struct{}

func (e errReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("stdin error")
}

func TestAddCommand_StdinError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords(env.pw)
	stdin = errReader{} // readLine("Name: ") will fail immediately

	err := env.run("add")
	if err == nil {
		t.Fatal("expected error when stdin fails")
	}
}

func TestGetCommand_StdinError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.setPasswords(env.pw)
	stdin = errReader{} // readLine("Search: ") will fail

	err := env.run("get") // no arg → prompts search
	if err == nil {
		t.Fatal("expected error when stdin fails")
	}
}

func TestEditCommand_StdinError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw", "https://github.com", "")
	env.setPasswords(env.pw)
	stdin = errReader{} // first readLine fails

	err := env.run("edit", "github")
	if err == nil {
		t.Fatal("expected error when stdin fails")
	}
}

func TestDeleteCommand_StdinError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw", "https://github.com", "")
	env.setPasswords(env.pw)
	stdin = errReader{} // readLine for confirmation fails

	err := env.run("delete", "github")
	if err == nil {
		t.Fatal("expected error when stdin fails")
	}
}

// --- URL retry in add ---

func TestAddCommand_URLRetry(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	pageInfoFetcher = func(url string) (*fetch.PageInfo, error) {
		return nil, fmt.Errorf("no fetch")
	}
	env.setPasswords(env.pw, []byte("pw"))
	// First URL is empty, second is valid
	env.setStdinLines("MyApp", "bob", "", "https://myapp.io", "notes")

	if err := env.run("add"); err != nil {
		t.Fatal(err)
	}
	v, _ := vault.Load(env.pwCopy())
	if v.Entries[0].URL != "https://myapp.io" {
		t.Errorf("expected URL https://myapp.io, got: %s", v.Entries[0].URL)
	}
}

// --- clipboard error ---

func TestGetCommand_ClipboardError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw", "https://github.com", "")
	env.setPasswords(env.pw)
	clipboardWriter = func(text string) error {
		return fmt.Errorf("clipboard unavailable")
	}

	err := env.run("get", "github")
	if err == nil {
		t.Fatal("expected error when clipboard fails")
	}
}

// --- mustLoadVault passwordReader error ---

func TestMustLoadVault_PasswordReaderError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	passwordReader = func(prompt string) ([]byte, error) {
		return nil, fmt.Errorf("tty unavailable")
	}

	err := env.run("list")
	if err == nil {
		t.Fatal("expected error when passwordReader fails")
	}
}

// TestExecute_Error verifies the error path in Execute() without calling os.Exit.
func TestExecute_Error(t *testing.T) {
	origExit := osExit
	var exitCode int
	osExit = func(code int) { exitCode = code }
	t.Cleanup(func() { osExit = origExit })

	// Capture stderr output from Execute.
	origStderr := stderr
	errBuf := &bytes.Buffer{}
	stderr = errBuf
	t.Cleanup(func() { stderr = origStderr })

	// Use an unknown command to trigger an error from rootCmd.Execute().
	SetArgs([]string{"__unknown_command__"})
	rootCmd.SilenceErrors = false
	rootCmd.SilenceUsage = true
	t.Cleanup(func() {
		rootCmd.SilenceErrors = false
		rootCmd.SilenceUsage = false
	})

	Execute()

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

// TestReadLine_EOF ensures the io.EOF break path in readLine is covered.
func TestReadLine_EOF(t *testing.T) {
	origStdin := stdin
	stdin = eofReader{}
	t.Cleanup(func() { stdin = origStdin })

	origStdout := stdout
	stdout = &bytes.Buffer{}
	t.Cleanup(func() { stdout = origStdout })

	result, err := readLine("prompt: ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string on EOF, got %q", result)
	}
}

// eofReader returns io.EOF on every Read.
type eofReader struct{}

func (e eofReader) Read(p []byte) (int, error) { return 0, io.EOF }

// TestSelectEntry_StdinError covers the readLine error path inside selectEntry.
func TestSelectEntry_StdinError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw1", "https://github.com", "")
	env.addEntry(t, "GitLab", "alice", "pw2", "https://gitlab.com", "")

	env.setPasswords(env.pw)
	// After the list is displayed, stdin returns an error instead of a selection.
	stdin = errReader{}

	err := env.run("get", "git")
	if err == nil {
		t.Fatal("expected error when stdin fails during entry selection")
	}
}

// TestAddCommand_SaveError covers the v.Save error return in the add command.
func TestAddCommand_SaveError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)

	// Make the vault directory read-only so Save fails.
	dir := filepath.Dir(vault.VaultPath())
	if err := os.Chmod(dir, 0500); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) }) //nolint:gosec

	pageInfoFetcher = func(url string) (*fetch.PageInfo, error) {
		return nil, fmt.Errorf("no fetch")
	}
	env.setPasswords(env.pw, []byte("pw"))
	env.setStdinLines("GitHub", "alice", "https://github.com", "notes")

	err := env.run("add")
	if err == nil {
		t.Fatal("expected error when vault save fails")
	}
}

// TestEditCommand_SaveError covers the v.Save error return in the edit command.
func TestEditCommand_SaveError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)
	env.addEntry(t, "GitHub", "alice", "pw", "https://github.com", "notes")

	dir := filepath.Dir(vault.VaultPath())
	if err := os.Chmod(dir, 0500); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) }) //nolint:gosec

	env.setPasswords(env.pw, []byte(""))
	env.setStdinLines("", "", "", "")

	err := env.run("edit", "github")
	if err == nil {
		t.Fatal("expected error when vault save fails")
	}
}

// TestPasswdCommand_PasswordReaderError covers passwordReader error paths in passwd.
func TestPasswdCommand_PasswordReaderError(t *testing.T) {
	env := newTestEnv(t)
	env.initVault(t)

	callCount := 0
	passwordReader = func(prompt string) ([]byte, error) {
		callCount++
		if callCount == 2 {
			return nil, fmt.Errorf("simulated tty error on second call")
		}
		return env.pwCopy(), nil
	}

	err := env.run("passwd")
	if err == nil {
		t.Fatal("expected error when second passwordReader call fails")
	}
}
