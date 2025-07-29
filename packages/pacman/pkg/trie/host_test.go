package trie

import "testing"

func TestHostTrieMatch(t *testing.T) {
	tree := NewHost[string]()

	tree.Insert("*.example.com", "wild-example")
	tree.Insert("*.test.example.com", "wild-test-example")
	tree.Insert("api.test.example.com", "literal-api-test")
	tree.Insert("*.api.test.example.com", "wild-api-test")
	tree.Insert("another.test.example.com", "literal-another-test")
	tree.Insert("*.a.b.com", "wild-a-b")
	tree.Insert("*.co.uk", "wild-co-uk")
	tree.Insert("*.Example.ORG", "wild-example-org")

	tests := []struct {
		host     string
		expectV  string
		expectOk bool
	}{
		{"foo.example.com", "wild-example", true},
		{"bar.test.example.com", "wild-test-example", true},
		{"api.test.example.com", "literal-api-test", true},
		{"sub.api.test.example.com", "wild-api-test", true},
		{"another.test.example.com", "literal-another-test", true},
		{"no.match.example.com", "wild-example", true},
		{"no.match.test.example.com", "wild-test-example", true},
		{"example.com", "", false},
		{"test.example.com", "wild-example", true},
		{"wrong.domain.net", "", false},
		{"x.b.com", "", false},                        // early miss
		{"x.a.b.com", "wild-a-b", true},               // deep match
		{"foo.co.uk", "wild-co-uk", true},             // multi-part tld
		{"FOO.EXAMPLE.ORG", "wild-example-org", true}, // case insensitivity
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.host, func(t *testing.T) {
			t.Parallel()
			val, ok := tree.Match(tc.host)
			if ok != tc.expectOk || val != tc.expectV {
				t.Errorf("got (%q, %v), want (%q, %v)", val, ok, tc.expectV, tc.expectOk)
			} else {
				t.Logf("✔ Match(%q) = (%q, %v)", tc.host, val, ok)
			}
		})
	}
}
