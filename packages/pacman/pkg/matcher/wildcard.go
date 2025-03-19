package matcher

import (
	"regexp"
	"strings"
)

// wildcardReplacer converts shell wildcards to regex.
var wildcardUnescaper = strings.NewReplacer(
	`\*`, ".*",
	`\?`, ".",
)

// NewWildcardMatcher compiles a shell-style wildcard pattern into a regex matcher.
func WildcardMatcher(expr string) (StringMatcher, error) {
	expr = regexp.QuoteMeta(expr)
	expr = wildcardUnescaper.Replace(expr)

	re, err := regexp.Compile("^" + expr + "$")
	if err != nil {
		return nil, err
	}

	return re, nil
}
