package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"paytabs/internal/errors"
)

// CardValidator validates card information.
type CardValidator struct{}

// NewCardValidator creates a new card validator.
func NewCardValidator() *CardValidator {
	return &CardValidator{}
}

// ValidateCard validates card number, expiry, and CVV.
func (v *CardValidator) ValidateCard(cardNumber, expiry, cvv string) error {
	// Remove spaces and dashes from card number
	cardNumber = strings.ReplaceAll(strings.ReplaceAll(cardNumber, " ", ""), "-", "")

	// Validate card number using Luhn algorithm
	if !v.validateLuhn(cardNumber) {
		return errors.ErrInvalidCard
	}

	// Validate expiry format (MM/YY)
	expiryRegex := regexp.MustCompile(`^(0[1-9]|1[0-2])/(\d{2})$`)
	if !expiryRegex.MatchString(expiry) {
		return errors.ErrInvalidCard
	}

	// Validate expiry is not in the past
	if !v.validateExpiry(expiry) {
		return errors.ErrInvalidCard
	}

	// Validate CVV (3-4 digits)
	cvvRegex := regexp.MustCompile(`^\d{3,4}$`)
	if !cvvRegex.MatchString(cvv) {
		return errors.ErrInvalidCard
	}

	return nil
}

// validateLuhn validates a card number using the Luhn algorithm.
func (v *CardValidator) validateLuhn(cardNumber string) bool {
	// Remove non-digits
	cardNumber = regexp.MustCompile(`\D`).ReplaceAllString(cardNumber, "")

	if len(cardNumber) < 13 || len(cardNumber) > 19 {
		return false
	}

	sum := 0
	isEven := false

	// Process from right to left
	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(cardNumber[i]))
		if err != nil {
			return false
		}

		if isEven {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isEven = !isEven
	}

	return sum%10 == 0
}

// validateExpiry validates that the expiry date is not in the past.
func (v *CardValidator) validateExpiry(expiry string) bool {
	parts := strings.Split(expiry, "/")
	if len(parts) != 2 {
		return false
	}

	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return false
	}

	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	// Convert YY to YYYY (assuming 20YY for years 00-99)
	if year < 100 {
		year += 2000
	}

	now := time.Now()
	expiryDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

	// Expiry should be at least the current month
	return expiryDate.After(now.AddDate(0, -1, 0))
}

// MaskCardNumber masks a card number, showing only last 4 digits.
func (v *CardValidator) MaskCardNumber(cardNumber string) string {
	cardNumber = strings.ReplaceAll(strings.ReplaceAll(cardNumber, " ", ""), "-", "")
	if len(cardNumber) < 4 {
		return "****"
	}
	return "****" + cardNumber[len(cardNumber)-4:]
}
