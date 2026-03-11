package vault

import (
	"bytes"
	"fmt"
	"testing"
)

func TestNewSalt(t *testing.T) {
	s1, err := newSalt()
	if err != nil {
		t.Fatal(err)
	}
	if len(s1) != saltLen {
		t.Fatalf("expected salt length %d, got %d", saltLen, len(s1))
	}

	s2, err := newSalt()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(s1, s2) {
		t.Fatal("two salts should not be equal")
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	password := []byte("testpassword")
	salt := make([]byte, saltLen)
	for i := range salt {
		salt[i] = byte(i)
	}

	k1 := deriveKey(password, salt)
	k2 := deriveKey(password, salt)

	if !bytes.Equal(k1, k2) {
		t.Fatal("same inputs should produce same key")
	}
	if len(k1) != argonKeyLen {
		t.Fatalf("expected key length %d, got %d", argonKeyLen, len(k1))
	}
}

func TestDeriveKey_DifferentSalt(t *testing.T) {
	password := []byte("testpassword")
	salt1 := make([]byte, saltLen)
	salt2 := make([]byte, saltLen)
	salt2[0] = 1

	k1 := deriveKey(password, salt1)
	k2 := deriveKey(password, salt2)
	if bytes.Equal(k1, k2) {
		t.Fatal("different salts should produce different keys")
	}
}

func TestDeriveKey_DifferentPassword(t *testing.T) {
	salt := make([]byte, saltLen)
	k1 := deriveKey([]byte("password1"), salt)
	k2 := deriveKey([]byte("password2"), salt)
	if bytes.Equal(k1, k2) {
		t.Fatal("different passwords should produce different keys")
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := make([]byte, argonKeyLen)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("hello, world! this is a secret message.")

	nonce, ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if len(nonce) != nonceLen {
		t.Fatalf("nonce length: want %d, got %d", nonceLen, len(nonce))
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := decrypt(key, nonce, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted mismatch: want %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptDecrypt_UniqueNonces(t *testing.T) {
	key := make([]byte, argonKeyLen)
	plaintext := []byte("test")

	nonce1, _, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	nonce2, _, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("nonces should be unique across calls")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key := make([]byte, argonKeyLen)
	nonce, ciphertext, err := encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	wrongKey := make([]byte, argonKeyLen)
	wrongKey[0] = 0xFF
	_, err = decrypt(wrongKey, nonce, ciphertext)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, argonKeyLen)
	nonce, ciphertext, err := encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[0] ^= 0xFF

	_, err = decrypt(key, nonce, tampered)
	if err == nil {
		t.Fatal("expected error with tampered ciphertext")
	}
}

func TestDecrypt_TruncatedCiphertext(t *testing.T) {
	key := make([]byte, argonKeyLen)
	nonce, ciphertext, err := encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = decrypt(key, nonce, ciphertext[:len(ciphertext)/2])
	if err == nil {
		t.Fatal("expected error with truncated ciphertext")
	}
}

func TestZeroBytes(t *testing.T) {
	b := []byte{1, 2, 3, 4, 5}
	ZeroBytes(b)
	for i, v := range b {
		if v != 0 {
			t.Fatalf("byte at index %d not zeroed: got %d", i, v)
		}
	}
}

func TestZeroBytes_Empty(t *testing.T) {
	ZeroBytes(nil)      // must not panic
	ZeroBytes([]byte{}) // must not panic
}

func TestEncrypt_BadKey(t *testing.T) {
	_, _, err := encrypt([]byte("tooshort"), []byte("plaintext"))
	if err == nil {
		t.Fatal("expected error for bad key length")
	}
}

func TestDecrypt_BadKey(t *testing.T) {
	_, err := decrypt([]byte("tooshort"), make([]byte, nonceLen), []byte("ciphertext"))
	if err == nil {
		t.Fatal("expected error for bad key length")
	}
}

func TestNewSalt_RandError(t *testing.T) {
	orig := randReader
	randReader = &errorReader{}
	defer func() { randReader = orig }()

	_, err := newSalt()
	if err == nil {
		t.Fatal("expected error when randReader fails")
	}
}

func TestEncrypt_NonceRandError(t *testing.T) {
	orig := randReader
	// Allow first read (for any prior usage) but we need to make the nonce generation fail.
	// Use a reader that succeeds for the first nonceLen bytes then fails.
	// Actually, the nonce read is inside encrypt after the cipher is created.
	// Use a reader that fails immediately.
	randReader = &errorReader{}
	defer func() { randReader = orig }()

	key := make([]byte, argonKeyLen)
	_, _, err := encrypt(key, []byte("test"))
	if err == nil {
		t.Fatal("expected error when nonce generation fails")
	}
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}
