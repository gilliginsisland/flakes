package matcher

type Literal string

func (m Literal) MatchString(s string) bool {
	return string(m) == s
}
