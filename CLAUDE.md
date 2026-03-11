# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**0pass** is a local password vault CLI application written in Go. It targets macOS, Linux, and Windows (no mobile). Users protect all stored credentials with a single master password.

## Build & Commands

```bash
go mod tidy               # sync dependencies
go build -o 0pass .       # build binary
go test ./...             # run all tests
go test ./... -run TestName  # run a single test
go vet ./...              # static analysis
```

## Architecture

### CLI
Use `cobra` for CLI command structure. Minimal v1 command set:

| Command | Description |
|---|---|
| `0pass init` | Create a new vault |
| `0pass add` | Add an entry (interactive prompt) |
| `0pass get <query>` | Fuzzy search; copy matched password to clipboard |
| `0pass list` | List all entries |
| `0pass edit <query>` | Edit an entry |
| `0pass delete <query>` | Delete an entry |
| `0pass passwd` | Change master password |

### Entry Schema
Each credential entry contains:

```go
type Entry struct {
    ID        string    // UUID
    Name      string    // human label, primary search target
    Username  string
    Password  string    // stored encrypted inside the vault
    URL       string    // mandatory; also used for smart context fetching
    Notes     string    // mandatory; may be auto-populated from the target URL
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

`url` and `notes` are mandatory. On `add`, the app should attempt to fetch the URL and populate `notes` with contextual info from the page (e.g., site title, description).

### Password Generator
Built-in generator for v1. Invoked when the user leaves the password field blank during `add` or `edit`. Should support configurable length and character sets (upper, lower, digits, symbols).

### Password Output
`get` copies the password to the system clipboard (not printed to stdout). Use `github.com/atotto/clipboard` for cross-platform clipboard support.

### Vault Storage
Single encrypted file at `~/.0pass/vault.dat`. Binary layout:

```
[4 bytes magic: "0PAS"] [1 byte version] [32 bytes Argon2id salt]
[12 bytes AES-GCM nonce] [N bytes GCM ciphertext+tag]
```

The plaintext is JSON-serialized `[]Entry`. GCM authentication tag is appended automatically by `crypto/cipher`.

### Key Derivation — critical
**Never** use the master password directly as an encryption key. Derive it:

```
key = Argon2id(password, salt, time=3, memory=64MB, threads=4, keyLen=32)
```

Use `golang.org/x/crypto/argon2`. The salt is stored in the vault file header; the derived key is never stored. On every unlock, re-derive the key from the password + stored salt.

### Encryption
AES-256-GCM via `crypto/aes` + `crypto/cipher`. GCM provides both confidentiality and integrity — any tampering causes decryption to fail.

### Fuzzy Search
Use `github.com/sahilm/fuzzy`. Match against `Name`, `URL`, `Username`. Present ranked results interactively when multiple entries match.

### In-memory hygiene
- Zero out sensitive byte slices (master password, derived key) immediately after use.
- Never log or print plaintext passwords or derived keys.

### Master Password Change
Re-encryption must be atomic:
1. Decrypt vault with old derived key
2. Generate a new random salt
3. Derive new key from new password + new salt
4. Encrypt vault with new key
5. Write to a temp file, then `os.Rename()` to vault path (atomic on all target platforms)

## Key Dependencies

| Purpose | Package |
|---|---|
| CLI framework | `github.com/spf13/cobra` |
| Key derivation | `golang.org/x/crypto/argon2` |
| Fuzzy search | `github.com/sahilm/fuzzy` |
| Clipboard | `github.com/atotto/clipboard` |
