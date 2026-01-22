package utils

import (
	"fmt"

	"github.com/google/uuid"
)

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
