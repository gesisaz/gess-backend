package secrets

import (
	"strings"
	"testing"
)

func TestLoadJWTSecret_empty(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	_, err := LoadJWTSecret()
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestLoadJWTSecret_tooShort(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("a", 31))
	_, err := LoadJWTSecret()
	if err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestLoadJWTSecret_ok(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	b, err := LoadJWTSecret()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 32 {
		t.Fatalf("len=%d", len(b))
	}
}
