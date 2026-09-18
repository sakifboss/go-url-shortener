package auth

import (
	"errors"
	"strings"
	"unicode"
)

// validatePassword enforces the minimum password policy for GoShort.
func validatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return errors.New("password is required")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	if len(password) > 72 {
		return errors.New("password must be at most 72 characters")
	}

	var hasLetter bool
	var hasNumber bool

	for _, char := range password {
		switch {
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	if !hasLetter {
		return errors.New("password must contain at least one letter")
	}

	if !hasNumber {
		return errors.New("password must contain at least one number")
	}

	return nil
}
