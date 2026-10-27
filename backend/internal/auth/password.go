package auth

import (
	"unicode"
	"unicode/utf8"

	"roadmates/backend/internal/apperrors"
)

// bcrypt accepts at most 72 bytes. Count characters for the minimum and reject
// oversized UTF-8 input rather than silently truncating it.
func ValidatePassword(password string) error {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return apperrors.ErrPasswordPolicy
	}
	var upper, lower, special bool
	for _, c := range password {
		if unicode.IsSpace(c) || unicode.IsControl(c) {
			return apperrors.ErrPasswordPolicy
		}
		upper = upper || unicode.IsUpper(c)
		lower = lower || unicode.IsLower(c)
		special = special || unicode.IsPunct(c) || unicode.IsSymbol(c)
	}
	if !upper || !lower || !special {
		return apperrors.ErrPasswordPolicy
	}
	return nil
}
