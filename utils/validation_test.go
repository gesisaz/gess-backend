package utils

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"", true},
		{"not-an-email", true},
		{"a@b", true},
		{"Bob <bob@example.com>", true},
		{"bob@example.com", false},
		{"  alice@test.org  ", false},
	}
	for _, tc := range tests {
		err := ValidateEmail(tc.in)
		if tc.wantErr && err == nil {
			t.Errorf("%q: want error", tc.in)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("%q: %v", tc.in, err)
		}
	}
}
