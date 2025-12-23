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
			ServicePack:  "26.1",
			Architecture: "arm64",
		},
		Policy: Policy{
			Location: "Default",
		},
		Device: Device{
			Protection:          "none",
			ProtectionVersion:   "5.1.8.122",
			ProtectionExtension: "4.3.3902.0",
		},
		Enforce: "success",
		PersonalFireWall: map[string]PersonalFireWall{
			"100386": {
				Exists:      true,
				Description: "Packet Filter (Mac)",
				Version:     "26.1",
				Enabled:     "ok",
			},
			"100250": {
				Exists:      true,
				Description: "CrowdStrike Falcon (Mac)",
				Version:     "7.29.20103.0",
				Enabled:     "ok",
			},
			"100022": {
				Exists:      true,
				Description: "Mac OS X Builtin Firewall (Mac)",
				Version:     "26.1",
				Enabled:     "ok",
			},
		},
		AntiMalware: map[string]AntiMalware{
			"100366": {
				Exists:      true,
				Description: "Xprotect (Mac)",
				Version:     "5325",
				Activescan:  "ok",
				LastUpdate:  545119,
				Timestamp:   int(time.Now().Unix()) - 545119,
			},
			"100250": {
				Exists:      true,
				Description: "CrowdStrike Falcon (Mac)",
				Version:     "7.29.20103.0",
				Activescan:  "ok",
				LastUpdate:  9447,
				Timestamp:   int(time.Now().Unix()) - 9447,
			},
			"100137": {
				Exists:      true,
				Description: "Gatekeeper (Mac)",
				Version:     "26.1",
				Activescan:  "ok",
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
