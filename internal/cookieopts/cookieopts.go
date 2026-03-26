package cookieopts

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Settings holds session cookie attributes derived from environment variables.
type Settings struct {
	Secure   bool
	SameSite http.SameSite
}

// Load reads COOKIE_SECURE and COOKIE_SAMESITE. SameSite "none" requires Secure true.
func Load() (Settings, error) {
	secure := strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true")
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SAMESITE")))
	if raw == "" {
		raw = "lax"
	}
	var ss http.SameSite
	switch raw {
	case "lax":
		ss = http.SameSiteLaxMode
	case "strict":
		ss = http.SameSiteStrictMode
	case "none":
		ss = http.SameSiteNoneMode
		if !secure {
			return Settings{}, fmt.Errorf("COOKIE_SAMESITE=none requires COOKIE_SECURE=true")
		}
	default:
		return Settings{}, fmt.Errorf("COOKIE_SAMESITE must be lax, strict, or none")
	}
	return Settings{Secure: secure, SameSite: ss}, nil
}

// SessionCookie builds an auth session cookie with shared attributes.
func SessionCookie(settings Settings, name, value string, expires time.Time) http.Cookie {
	return http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expires,
		HttpOnly: true,
		SameSite: settings.SameSite,
		Secure:   settings.Secure,
		Path:     "/",
	}
}
