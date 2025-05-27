package hostscan

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/gilliginsisland/pacman/pkg/netutil"
)

type Client struct {
	host   string
	client *http.Client
}

func NewClient(host string, fingerprint []byte) *Client {
	return &Client{
		host:   host,
		client: netutil.NewHPKPClient(fingerprint),
	}
}

func (c *Client) GetManifest() (*Manifest, error) {
	resp, err := c.client.Get(fmt.Sprintf("https://%s/CACHE/sdesktop/data.xml", c.host))
	if err != nil {
		return nil, fmt.Errorf("Error fetching hostscan manifest: %w", err)
	}
	defer resp.Body.Close()

	m, err := ReadManifest(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing hostscan manifest: %v", err)
	}
	return m, nil
}

func (c *Client) GetToken(ticket, stub string) (string, error) {
	u := fmt.Sprintf("https://%s/+CSCOE+/sdesktop/token.xml?ticket=%s&stub=%s", c.host, ticket, stub)
	resp, err := c.client.Get(u)
	if err != nil {
		return "", fmt.Errorf("Error fetching hostscan token: %w", err)
	}
	defer resp.Body.Close()

	t := struct {
		XMLName xml.Name `xml:"hostscan"`
		Token   string   `xml:"token"`
	}{}
	if err := xml.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", fmt.Errorf("Error parsing hostscan token: %w", err)
	}

	return t.Token, nil
}

func (c *Client) PostReport(report *Report, token string) (*http.Response, error) {
	data := report.String()

	u := fmt.Sprintf("https://%s/+CSCOE+/sdesktop/scan.xml?reusebrowser=1", c.host)
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{
		Name:  "sdesktop",
		Value: token,
	})
	req.ContentLength = int64(len(data))
	req.TransferEncoding = []string{"identity"}
	return c.client.Do(req)
}
