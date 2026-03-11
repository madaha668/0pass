package generator

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.Length != 20 {
		t.Errorf("default length: want 20, got %d", opts.Length)
	}
	if !opts.Upper {
		t.Error("default Upper should be true")
	}
	if !opts.Digits {
		t.Error("default Digits should be true")
	}
	if !opts.Symbols {
		t.Error("default Symbols should be true")
	}
}

func TestGenerate_DefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	pw, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(pw) != 20 {
		t.Errorf("expected length 20, got %d", len(pw))
	}
}

func TestGenerate_Length(t *testing.T) {
	for _, l := range []int{1, 8, 32, 64} {
		pw, err := Generate(Options{Length: l, Upper: true, Digits: true, Symbols: true})
		if err != nil {
			t.Fatalf("length %d: %v", l, err)
		}
		if len(pw) != l {
			t.Errorf("length %d: got %d", l, len(pw))
		}
	}
}

func TestGenerate_ZeroLength_ReturnsError(t *testing.T) {
	_, err := Generate(Options{Length: 0})
	if err == nil {
		t.Fatal("expected error for zero length")
	}
}

func TestGenerate_LowerOnly(t *testing.T) {
	opts := Options{Length: 100, Upper: false, Digits: false, Symbols: false}
	pw, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range pw {
		if !unicode.IsLower(c) {
			t.Fatalf("expected only lowercase, got %q in %q", c, pw)
		}
	}
}

func TestGenerate_WithUpper(t *testing.T) {
	opts := Options{Length: 200, Upper: true, Digits: false, Symbols: false}
	pw, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.ContainsAny(pw, uppers) {
		t.Error("expected at least one uppercase character")
	}
}

func TestGenerate_WithDigits(t *testing.T) {
	opts := Options{Length: 200, Upper: false, Digits: true, Symbols: false}
	pw, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.ContainsAny(pw, digits) {
		t.Error("expected at least one digit")
	}
}

func TestGenerate_WithSymbols(t *testing.T) {
	opts := Options{Length: 200, Upper: false, Digits: false, Symbols: true}
	pw, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.ContainsAny(pw, symbols) {
		t.Error("expected at least one symbol")
	}
}

func TestGenerate_AllOptions(t *testing.T) {
	opts := Options{Length: 200, Upper: true, Digits: true, Symbols: true}
	pw, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.ContainsAny(pw, uppers) {
		t.Error("expected uppercase")
	}
	if !strings.ContainsAny(pw, digits) {
		t.Error("expected digit")
	}
	if !strings.ContainsAny(pw, symbols) {
		t.Error("expected symbol")
	}
}

func TestGenerate_Randomness(t *testing.T) {
	opts := DefaultOptions()
	pw1, _ := Generate(opts)
	pw2, _ := Generate(opts)
	if pw1 == pw2 {
		t.Error("two generated passwords should differ (extremely unlikely to be equal)")
	}
}

func TestGenerate_OnlyValidChars(t *testing.T) {
	opts := Options{Length: 500, Upper: true, Digits: true, Symbols: true}
	pw, err := Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	charset := lowers + uppers + digits + symbols
	for _, c := range pw {
		if !strings.ContainsRune(charset, c) {
			t.Fatalf("unexpected character %q in generated password", c)
		}
	}
}

func TestGenerate_RandError(t *testing.T) {
	orig := randReader
	randReader = &errorReader{}
	defer func() { randReader = orig }()

	_, err := Generate(Options{Length: 5})
	if err == nil {
		t.Fatal("expected error when rand fails")
	}
}

// errorReader always returns an error.
type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("simulated error")
}
