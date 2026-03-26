package cookieopts

import (
	"net/http"
	"testing"
	"time"
)

func TestLoad_sameSiteNoneRequiresSecure(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_SAMESITE", "none")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when SameSite=none and Secure=false")
	}
}

func TestLoad_sameSiteNoneWithSecure(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("COOKIE_SAMESITE", "none")
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !s.Secure || s.SameSite != http.SameSiteNoneMode {
		t.Fatalf("%+v", s)
	}
}

func TestLoad_defaultsLax(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("COOKIE_SAMESITE", "")
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.Secure {
		t.Fatal("expected Secure false by default")
	}
	if s.SameSite != http.SameSiteLaxMode {
		t.Fatalf("got %v", s.SameSite)
	}
}

func TestSessionCookie(t *testing.T) {
	s := Settings{Secure: false, SameSite: http.SameSiteLaxMode}
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c := SessionCookie(s, "token", "abc", exp)
	if c.Name != "token" || c.Value != "abc" || !c.HttpOnly || c.Path != "/" {
		t.Fatalf("%+v", c)
	}
}
