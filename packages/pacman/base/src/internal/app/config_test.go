package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfigFileOptions(t *testing.T) {
	for _, tt := range []struct {
		name    string
		data    string
		want    string
		wantErr bool
	}{
		{"yaml number", "proxies:\n  test:\n    options:\n      timeout: 1800\n", "1800", false},
		{"yaml string", "proxies:\n  test:\n    options:\n      timeout: \"1800\"\n", "1800", false},
		{"yaml zero", "proxies:\n  test:\n    options:\n      timeout: 0\n", "0", false},
		{"json number", `{"proxies":{"test":{"options":{"timeout":1800}}}}`, "1800", false},
		{"json string", `{"proxies":{"test":{"options":{"timeout":"1800"}}}}`, "1800", false},
		{"json zero", `{"proxies":{"test":{"options":{"timeout":0}}}}`, "0", false},
		{"URL string", `{"proxies":{"test":"socks5://localhost:1080?timeout=1800"}}`, "1800", false},
		{"yaml list", "proxies:\n  test:\n    options:\n      timeout: [1800]\n", "", true},
		{"json object", `{"proxies":{"test":{"options":{"timeout":{"seconds":1800}}}}}`, "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "config")
			if err := os.WriteFile(file, []byte(tt.data), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := ParseConfigFile(Path(file))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected invalid option value to be rejected")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.Proxies["test"].Query().Get("timeout"); got != tt.want {
				t.Errorf("timeout = %q, want %q", got, tt.want)
			}
		})
	}
}
