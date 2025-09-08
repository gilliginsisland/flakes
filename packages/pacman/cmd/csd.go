package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gilliginsisland/pacman/pkg/env"
	"github.com/gilliginsisland/pacman/pkg/openconnect/hostscan"
)

func init() {
	parser.AddCommand("csd", "Run the csd hostscan", "Sends the hostscan report", &CSDCommand{})
}

type CSDEnv struct {
	Hostname string `env:"CSD_HOSTNAME"`
	Token    string `env:"CSD_TOKEN"`
	SHA256   string `env:"CSD_SHA256"`
}

type CSDCommand struct{}

func (c *CSDCommand) Execute(args []string) error {
	var e CSDEnv
	err := env.Unmarshal(&e, os.Environ())
	if err != nil {
		return fmt.Errorf("error parsing env: %w", err)
	}

	hash, err := base64.StdEncoding.DecodeString(e.SHA256)
	if err != nil {
		return fmt.Errorf("Invalid CSD_SHA256: %w", err)
	}

	client := hostscan.NewClient(e.Hostname, hash)

	manifest, err := client.GetManifest()
	if err != nil {
		return fmt.Errorf("Error getting hostscan policy: %w", err)
	}

	rep := hostscan.NewMockScanner(manifest).Scan()

	res, err := client.PostReport(rep, e.Token)
	if err != nil {
		return fmt.Errorf("Error posting hostscan report: %w", err)
	}
	defer res.Body.Close()

	if txt, err := io.ReadAll(res.Body); err != nil {
		return fmt.Errorf("Error reading csd POST response: %w", err)
	} else if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Hostscan server returned error: %s", string(txt))
	} else {
		log.Printf("Hostscan completed successfully\n%s", string(txt))
	}

	return nil
}
