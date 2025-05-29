package hostscan

import "time"

type Scanner interface {
	Scan() *Report
}

type mockScanner struct {
	manifest *Manifest
}

func (s *mockScanner) Scan() *Report {
	r := Report{
		OS: OS{
			Version:      "Darwin",
			ServicePack:  "24.5.0",
			Architecture: "arm64",
		},
		Policy: Policy{
			Location: "Default",
		},
		Device: Device{
			Protection:          "none",
			ProtectionVersion:   "4.10.01094",
			ProtectionExtension: "4.3.1858.0",
		},
		Enforce: "success",
		PersonalFireWall: map[string]PersonalFireWall{
			"100022": {
				Exists:      true,
				Description: "Mac OS X Builtin Firewall (Mac)",
				Version:     "11.5.1",
				Enabled:     "ok",
			},
			"100194": {
				Exists:      true,
				Description: "McAfee Endpoint Security for Mac (Mac)",
				Version:     "10.7.7",
				Enabled:     "ok",
			},
		},
		AntiMalware: map[string]AntiMalware{
			"100137": {
				Exists:      true,
				Description: "Gatekeeper (Mac)",
				Version:     "11.5.1",
				Activescan:  "ok",
			},
			"100194": {
				Exists:      true,
				Description: "McAfee Endpoint Security for Mac (Mac)",
				Version:     "10.7.7",
				Activescan:  "ok",
				LastUpdate:  56266,
				Timestamp:   int(time.Now().Unix()) - 56266,
			},
		},
		Files:     map[string]File{},
		Processes: map[string]Process{},
	}

	for _, i := range s.manifest.Fields {
		switch i.Type {
		case "File":
			r.Files[i.Label] = File{
				Name:         i.Value,
				Path:         i.Value,
				Exists:       true,
				LastModified: 39115924,
				Timestamp:    int(time.Now().Unix()) - 39115924,
			}
		case "Process":
			r.Processes[i.Label] = Process{
				Name:   i.Value,
				Exists: true,
			}
		}
	}

	return &r
}

func NewMockScanner(m *Manifest) Scanner {
	return &mockScanner{
		manifest: m,
	}
}
