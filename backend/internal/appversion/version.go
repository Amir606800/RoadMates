package appversion

import (
	"regexp"

	"golang.org/x/mod/semver"
)

var fullVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func Valid(value string) bool {
	return fullVersion.MatchString(value) && semver.IsValid("v"+value)
}

// Compare receives validated versions and follows SemVer precedence rules.
func Compare(a, b string) int { return semver.Compare("v"+a, "v"+b) }
