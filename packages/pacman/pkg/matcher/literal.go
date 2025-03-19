package matcher

type LiteralMatcher string

func (m LiteralMatcher) MatchString(s string) bool {
	return string(m) == s
}
