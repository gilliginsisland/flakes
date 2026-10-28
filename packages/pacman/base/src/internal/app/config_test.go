package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigPassword(t *testing.T) {
	for _, tc := range []struct{ name, value, want string }{
		{"plain", "ABC.5jTk^25d5df", "ABC.5jTk^25d5df"},
		{"quoted punctuation", "'!C3&%xBfZ2o7h&A@'", "!C3&%xBfZ2o7h&A@"},
		{"quotes and backslash", `'a''b"c\d: #@/?%'`, `a'b"c\d: #@/?%`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "config")
			data := "proxies:\n  vpn:\n    protocol: anyconnect\n    host: vpn.example.com\n    username: user\n    password: " + tc.value + "\n"
			if err := os.WriteFile(file, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := ParseConfigFile(Path(file))
			if err != nil {
				t.Fatal(err)
			}
			u := cfg.Proxies["vpn"]
			got, ok := u.User.Password()
			if !ok || got != tc.want {
				t.Fatalf("password was not preserved")
			}
			var roundTrip URL
			if err := roundTrip.UnmarshalBinary([]byte(u.String())); err != nil {
				t.Fatal(err)
			}
			got, ok = roundTrip.User.Password()
			if !ok || got != tc.want {
				t.Fatal("URL round trip changed password")
			}
		})
	}
}

func TestParseConfigInvalidPassword(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(file, []byte("proxies:\n  vpn:\n    password: !C3&%xBfZ2o7h&A@\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := ParseConfigFile(Path(file))
	if err == nil {
		t.Fatal("expected invalid YAML to be rejected")
	}
	if !strings.Contains(err.Error(), "single quotes") {
		t.Fatal("missing quoting guidance")
	}
	if strings.Contains(err.Error(), "!C3&%xBfZ2o7h&A@") {
		t.Fatal("error exposed password")
	}
}
