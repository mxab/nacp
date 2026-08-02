package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

type caOpts struct {
	Name                string
	Days                int
	PermittedDNSDomains []string
}

type certOpts struct {
	Signer      crypto.Signer
	CA          string
	Name        string
	Days        int
	DNSNames    []string
	IPAddresses []net.IP
	ExtKeyUsage []x509.ExtKeyUsage
}

func serialNumber() (*big.Int, error) {
	return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
}

func encodeKey(key *ecdsa.PrivateKey) (string, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})), nil
}

func encodeCert(der []byte) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// generateCA creates a self-signed CA certificate and its private key, both
// PEM encoded.
func generateCA(opts caOpts) (string, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	sn, err := serialNumber()
	if err != nil {
		return "", "", err
	}

	name := opts.Name
	if name == "" {
		name = fmt.Sprintf("NACP Test CA %s", sn)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          sn,
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().AddDate(0, 0, opts.Days),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
	}
	if len(opts.PermittedDNSDomains) > 0 {
		tmpl.PermittedDNSDomainsCritical = true
		tmpl.PermittedDNSDomains = opts.PermittedDNSDomains
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return "", "", err
	}

	keyPEM, err := encodeKey(key)
	if err != nil {
		return "", "", err
	}
	return encodeCert(der), keyPEM, nil
}

// parseSigner decodes a PEM encoded EC private key into a crypto.Signer.
func parseSigner(pemValue string) (crypto.Signer, error) {
	block, _ := pem.Decode([]byte(pemValue))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

// generateCert issues a leaf certificate signed by the given CA.
func generateCert(opts certOpts) (string, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	sn, err := serialNumber()
	if err != nil {
		return "", "", err
	}

	caBlock, _ := pem.Decode([]byte(opts.CA))
	if caBlock == nil {
		return "", "", fmt.Errorf("no PEM block found in CA")
	}
	parent, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return "", "", err
	}

	tmpl := &x509.Certificate{
		SerialNumber:          sn,
		Subject:               pkix.Name{CommonName: opts.Name},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().AddDate(0, 0, opts.Days),
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           opts.ExtKeyUsage,
		DNSNames:              opts.DNSNames,
		IPAddresses:           opts.IPAddresses,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, key.Public(), opts.Signer)
	if err != nil {
		return "", "", err
	}

	keyPEM, err := encodeKey(key)
	if err != nil {
		return "", "", err
	}
	return encodeCert(der), keyPEM, nil
}

// verifyCert checks that the leaf certificate chains to the CA and is valid for
// the given name.
func verifyCert(caPEM, certPEM, name string) error {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(caPEM)) {
		return fmt.Errorf("failed to parse CA certificate")
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return fmt.Errorf("no PEM block found in certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}

	_, err = cert.Verify(x509.VerifyOptions{
		DNSName: name,
		Roots:   roots,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
	})
	return err
}
