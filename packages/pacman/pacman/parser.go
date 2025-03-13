package pacman

import (
	"regexp"
	"strings"
)

var wildcardReplacer = strings.NewReplacer(
	".", "\\.",
	"?", ".?",
	"*", ".*",
)

// compileWildcard will convert a shell style wildcard to regex matcher
func compileWildcard(expr string) (*regexp.Regexp, error) {
	expr = wildcardReplacer.Replace(expr)
	return regexp.Compile("^" + expr + "$")
}
