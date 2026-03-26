package jwtutil

import (
	"testing"
	"time"

	"gess-backend/models"

	"github.com/google/uuid"
)

func TestSignAndParse(t *testing.T) {
	Init([]byte("01234567890123456789012345678901"))
	id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	u := models.User{ID: id, Username: "alice", Role: models.UserRoleUser}
	tok, _, err := SignToken(u, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Username != "alice" || claims.Subject != id.String() {
		t.Fatalf("%+v", claims)
	}
}
