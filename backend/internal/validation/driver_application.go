package validation

import (
	"regexp"
	"unicode"
	"unicode/utf8"
)

var pinPattern = regexp.MustCompile(`^[A-Z0-9]{7}$`)
var platePattern = regexp.MustCompile(`^[\pL\pN][\pL\pN -]{2,14}$`)

func PIN(value string) bool   { return pinPattern.MatchString(value) }
func Plate(value string) bool { return platePattern.MatchString(value) }
func Text(value string, max int, allowNewline bool) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return false
	}
	for _, c := range value {
		if unicode.IsControl(c) && !(allowNewline && c == '\n') {
			return false
		}
	}
	return true
}
