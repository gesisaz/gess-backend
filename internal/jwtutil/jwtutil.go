package jwtutil

import (
	"fmt"
	"sync"
	"time"

	"gess-backend/models"

	"github.com/golang-jwt/jwt/v5"
)

var (
	keyMu sync.RWMutex
	key   []byte
)

// Init sets the HMAC key used to sign and verify JWTs. Call once at process startup.
func Init(secret []byte) {
	keyMu.Lock()
	defer keyMu.Unlock()
	key = secret
}

func signingKey() ([]byte, error) {
	keyMu.RLock()
	defer keyMu.RUnlock()
	if len(key) == 0 {
		return nil, fmt.Errorf("jwtutil: Init was not called with a secret")
	}
	return key, nil
}

// Claims matches the JWT payload used for session tokens.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// SignToken issues an HS256 JWT for the given user.
func SignToken(user models.User, ttl time.Duration) (token string, expires time.Time, err error) {
	sk, err := signingKey()
	if err != nil {
		return "", time.Time{}, err
	}
	expires = time.Now().Add(ttl)
	claims := &Claims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
			Subject:   user.ID.String(),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(sk)
	if err != nil {
		return "", time.Time{}, err
	}
	return s, expires, nil
}

// ParseToken validates the token and returns claims.
func ParseToken(tokenString string) (*Claims, error) {
	sk, err := signingKey()
	if err != nil {
		return nil, err
	}
	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return sk, nil
	})
	if err != nil || !tkn.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
