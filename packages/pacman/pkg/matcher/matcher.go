package matcher

import (
	"regexp"
	"strings"
)

type StringMatcher interface {
	MatchString(string) bool
}

// CompileMatcher will take a textual pattern and return the
// correct StringMatcher type based on the format
func CompileMatcher(s string) (StringMatcher, error) {
	switch {
	case strings.HasPrefix(s, "/") && strings.HasSuffix(s, "/"):
		return regexp.Compile(s[1 : len(s)-1])
	case cidrRegex.MatchString(s):
		return NewCIDRMatcher(s)
	case strings.ContainsAny(s, "*?"):
		return WildcardMatcher(s)
	default:
		return LiteralMatcher(s), nil
	}
}
