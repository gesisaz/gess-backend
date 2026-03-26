package utils

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const maxEmailLen = 254

// ValidateEmail checks that s is a non-empty RFC 5322-style address (via net/mail)
// with reasonable length. It rejects display-name forms ("Name <a@b.com>").
func ValidateEmail(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("email is required")
	}
	if utf8.RuneCountInString(s) > maxEmailLen {
		return fmt.Errorf("email is too long")
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return fmt.Errorf("invalid email address")
	}
	if addr.Name != "" {
		return fmt.Errorf("invalid email address: use addr-spec only, not display name form")
	}
	at := strings.LastIndex(addr.Address, "@")
	if at <= 0 || at == len(addr.Address)-1 {
		return fmt.Errorf("invalid email address")
	}
	host := addr.Address[at+1:]
	if !strings.Contains(host, ".") && host != "localhost" {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

// ValidateUUID validates a UUID string
func ValidateUUID(uuidStr string) (uuid.UUID, error) {
	if uuidStr == "" {
		return uuid.Nil, fmt.Errorf("UUID cannot be empty")
	}
	
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID format: %s", uuidStr)
	}
	
	return parsed, nil
}

// ValidateRequired checks if a string field is not empty
func ValidateRequired(field, fieldName string) error {
	if field == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidatePositiveFloat checks if a float is positive
func ValidatePositiveFloat(value float64, fieldName string) error {
	if value <= 0 {
		return fmt.Errorf("%s must be greater than 0", fieldName)
	}
	return nil
}

// ValidateNonNegativeInt checks if an integer is non-negative
func ValidateNonNegativeInt(value int, fieldName string) error {
	if value < 0 {
		return fmt.Errorf("%s must be 0 or greater", fieldName)
	}
	return nil
}

// ValidatePriceRange checks if min and max prices are valid
func ValidatePriceRange(minPrice, maxPrice *float64) error {
	if minPrice != nil && *minPrice < 0 {
		return fmt.Errorf("minimum price cannot be negative")
	}
	if maxPrice != nil && *maxPrice < 0 {
		return fmt.Errorf("maximum price cannot be negative")
	}
	if minPrice != nil && maxPrice != nil && *minPrice > *maxPrice {
		return fmt.Errorf("minimum price cannot be greater than maximum price")
	}
	return nil
}

// Cosmetics-specific validations

// ValidSizeUnits are the accepted size units for cosmetics products
var ValidSizeUnits = map[string]bool{
	"ml":  true,
	"oz":  true,
	"g":   true,
	"kg":  true,
	"l":   true,
	"floz": true,
}

// ValidSkinTypes are the accepted skin type values
var ValidSkinTypes = map[string]bool{
	"dry":         true,
	"oily":        true,
	"sensitive":   true,
	"normal":      true,
	"combination": true,
	"mature":      true,
	"all":         true,
	"very dry":    true,
}

// ValidApplicationAreas are the accepted application area values
var ValidApplicationAreas = map[string]bool{
	"body":  true,
	"face":  true,
	"hands": true,
	"feet":  true,
	"hair":  true,
	"nails": true,
}

// ValidateSizeUnit checks if a size unit is valid
func ValidateSizeUnit(unit string) error {
	if unit == "" {
		return nil // Optional field
	}
	
	unit = strings.ToLower(strings.TrimSpace(unit))
	if !ValidSizeUnits[unit] {
		return fmt.Errorf("invalid size unit: %s. Must be one of: ml, oz, g, kg, l, floz", unit)
	}
	return nil
}

// ValidateSkinTypes checks if skin types are valid
func ValidateSkinTypes(skinTypes []string) error {
	if len(skinTypes) == 0 {
		return nil // Optional field
	}
	
	for _, skinType := range skinTypes {
		skinType = strings.ToLower(strings.TrimSpace(skinType))
		if !ValidSkinTypes[skinType] {
			return fmt.Errorf("invalid skin type: %s. Must be one of: dry, oily, sensitive, normal, combination, mature, all, very dry", skinType)
		}
	}
	return nil
}

// ValidateApplicationArea checks if application area is valid
func ValidateApplicationArea(area string) error {
	if area == "" {
		return nil // Optional field
	}
	
	area = strings.ToLower(strings.TrimSpace(area))
	if !ValidApplicationAreas[area] {
		return fmt.Errorf("invalid application area: %s. Must be one of: body, face, hands, feet, hair, nails", area)
	}
	return nil
}

// ValidateRating checks if rating is within valid range
func ValidateRating(rating float64) error {
	if rating < 0 || rating > 5 {
		return fmt.Errorf("rating must be between 0 and 5")
	}
	return nil
}

// ValidateSKU validates SKU format (basic validation)
func ValidateSKU(sku string) error {
	if sku == "" {
		return nil // Optional field
	}
	
	if len(sku) < 3 {
		return fmt.Errorf("SKU must be at least 3 characters long")
	}
	
	if len(sku) > 100 {
		return fmt.Errorf("SKU must be at most 100 characters long")
	}
	
	return nil
}
