package utils

import "strings"

// NormalizeMpesaPhone returns phone in 254XXXXXXXXX format (12 chars: 254 + 9 digits) or empty string if invalid.
func NormalizeMpesaPhone(phone string) string {
	s := strings.TrimSpace(phone)
	s = strings.TrimPrefix(s, "+")
	if strings.HasPrefix(s, "0") {
		s = "254" + s[1:]
	} else if !strings.HasPrefix(s, "254") {
		s = "254" + s
	}
	if len(s) != 12 {
		return ""
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return s
}
