package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateBook(t *testing.T) {
	source := []byte("# Source title\n\nIntro [reference][ref].\n\n" +
		"## Setup\n\nStart here.\n\n### Rules\n\n#### Matching\n\n" +
		"<a id=\"wildcards\"></a>\n\n" +
		"| Status | Meaning |\n| --- | --- |\n| 🟡\u00a0Connecting | Please wait |\n\n" +
		"```yaml\n## not a page\nhost: example.com\n```\n\n" +
		"### Next\n\nSee [reference][ref].\n\n[ref]: https://example.com\n")
	dir := t.TempDir()
	if err := generate(source, dir, "Example & Help", "example.help"); err != nil {
		t.Fatal(err)
	}
	checks := map[string][]string{
		"index.html": {
			`<meta name="AppleTitle" content="example.help">`,
			`Example &amp; Help`, `href="https://example.com"`,
			`href="setup.html"`, `href="rules.html"`, `href="next.html"`,
		},
		"rules.html": {
			`<a href="setup.html">Setup</a>`, `<h2>Matching</h2>`,
			`<a id="wildcards"></a>`, `<table>`, "🟡\u00a0Connecting",
			`class="language-yaml"`, "## not a page\nhost: example.com",
			`href="rules.html" aria-current="page"`,
		},
		"next.html": {`href="https://example.com"`},
	}
	for name, fragments := range checks {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(string(data), fragment) {
				t.Errorf("%s missing %q", name, fragment)
			}
		}
		if strings.Contains(string(data), "<script") {
			t.Errorf("%s contains runtime JavaScript", name)
		}
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 6 { // Four pages and two stylesheets; headings inside code stay in code.
		t.Errorf("got %d output files, want 6", len(files))
	}
}
