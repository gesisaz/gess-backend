package secrets

import (
	"fmt"
	"os"
	"strings"
)

const minJWTSecretLen = 32

// LoadJWTSecret reads JWT_SECRET from the environment (trimmed). It rejects empty
// or short secrets so weak defaults cannot be used in production.
func LoadJWTSecret() ([]byte, error) {
	s := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if s == "" {
		return nil, fmt.Errorf("JWT_SECRET is required and must not be empty")
	}
	if len(s) < minJWTSecretLen {
		return nil, fmt.Errorf("JWT_SECRET must be at least %d bytes", minJWTSecretLen)
	}
	return []byte(s), nil
}
