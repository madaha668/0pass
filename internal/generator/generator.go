package generator

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	lowers  = "abcdefghijklmnopqrstuvwxyz"
	uppers  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits  = "0123456789"
	symbols = "!@#$%^&*()-_=+[]{}|;:,.<>?"
)

// Options controls password generation.
type Options struct {
	Length  int
	Upper   bool
	Digits  bool
	Symbols bool
}

// DefaultOptions returns sensible defaults: 20 chars with all character classes.
func DefaultOptions() Options {
	return Options{
		Length:  20,
		Upper:   true,
		Digits:  true,
		Symbols: true,
	}
}

// Generate produces a cryptographically random password according to opts.
func Generate(opts Options) (string, error) {
	charset := lowers
	if opts.Upper {
		charset += uppers
	}
	if opts.Digits {
		charset += digits
	}
	if opts.Symbols {
		charset += symbols
	}

	if len(charset) == 0 {
		return "", fmt.Errorf("no character classes selected")
	}
	if opts.Length <= 0 {
		return "", fmt.Errorf("password length must be positive")
	}

	charsetLen := big.NewInt(int64(len(charset)))
	result := make([]byte, opts.Length)
	for i := range result {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("generating random character: %w", err)
		}
		result[i] = charset[n.Int64()]
	}

	return string(result), nil
}
