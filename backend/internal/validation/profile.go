package validation

import (
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

func Email(value string) bool {
	if len(value) > 254 || strings.ContainsAny(value, "\r\n") {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && address.Name == ""
}

func Name(value string) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > 60 {
		return false
	}
	for _, c := range value {
		if !unicode.IsLetter(c) && !unicode.IsMark(c) && c != ' ' && c != '-' && c != '\'' && c != '’' {
			return false
		}
	}
	return true
}

func Phone(value string) bool {
	if len(value) < 7 || len(value) > 25 {
		return false
	}
	digits := 0
	for i, c := range value {
		if c >= '0' && c <= '9' {
			digits++
			continue
		}
		if c != ' ' && c != '-' && c != '(' && c != ')' && !(c == '+' && i == 0) {
			return false
		}
	}
	return digits >= 7 && digits <= 15
}
