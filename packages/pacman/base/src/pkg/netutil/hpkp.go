package netutil

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
)

type Fingerprint []byte

func (fp Fingerprint) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	for _, raw := range rawCerts {
		if cert, err := x509.ParseCertificate(raw); err != nil {
			return err
		} else if der, err := x509.MarshalPKIXPublicKey(cert.PublicKey); err != nil {
			return err
		} else if hash := sha256.Sum256(der); bytes.Compare(hash[0:], fp) == 0 {
			return nil
		}
	}
	return errors.New("Pin validation failed")
}

func NewHPKPClient(fp Fingerprint) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify:    true,
				VerifyPeerCertificate: fp.VerifyPeerCertificate,
			},
		},
	}
}
