/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

const (
	certRSABits   = 2048
	certValidDays = 3650 // ~10 years, matching generate_certs.sh
)

// certBundle holds PEM-encoded certificate material.
type certBundle struct {
	CACertPEM   []byte
	NodeCertPEM []byte
	NodeKeyPEM  []byte
}

// generateCertBundle creates a self-signed CA and a CA-signed node certificate.
// The node key is encoded in PKCS#8 format, matching the reference script.
// No SAN extension is included (only needed for domain-based access).
func generateCertBundle() (*certBundle, error) {
	// ---- CA key + self-signed certificate ----
	caKey, err := rsa.GenerateKey(rand.Reader, certRSABits)
	if err != nil {
		return nil, fmt.Errorf("generate CA key: %w", err)
	}

	caSerialNumber, err := randomSerialNumber()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	caTemplate := &x509.Certificate{
		SerialNumber: caSerialNumber,
		Subject: pkix.Name{
			Country:            []string{"CN"},
			Organization:       []string{"INFINI"},
			OrganizationalUnit: []string{"Easysearch"},
			CommonName:         "Easysearch CA",
		},
		NotBefore:             now,
		NotAfter:              now.AddDate(0, 0, certValidDays),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("create CA certificate: %w", err)
	}

	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	// Parse back to use as signing parent.
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	// ---- Node key + CA-signed certificate ----
	nodeKey, err := rsa.GenerateKey(rand.Reader, certRSABits)
	if err != nil {
		return nil, fmt.Errorf("generate node key: %w", err)
	}

	nodeSerialNumber, err := randomSerialNumber()
	if err != nil {
		return nil, err
	}

	nodeTemplate := &x509.Certificate{
		SerialNumber: nodeSerialNumber,
		Subject: pkix.Name{
			Country:            []string{"CN"},
			Organization:       []string{"INFINI"},
			OrganizationalUnit: []string{"Easysearch"},
			CommonName:         "Easysearch Node",
		},
		NotBefore: now,
		NotAfter:  now.AddDate(0, 0, certValidDays),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
	}

	nodeCertDER, err := x509.CreateCertificate(rand.Reader, nodeTemplate, caCert, &nodeKey.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("create node certificate: %w", err)
	}

	nodeCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: nodeCertDER})

	// Encode node private key as PKCS#8 (matching generate_certs.sh -nocrypt).
	nodeKeyPKCS8, err := x509.MarshalPKCS8PrivateKey(nodeKey)
	if err != nil {
		return nil, fmt.Errorf("marshal node key to PKCS#8: %w", err)
	}
	nodeKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: nodeKeyPKCS8})

	return &certBundle{
		CACertPEM:   caCertPEM,
		NodeCertPEM: nodeCertPEM,
		NodeKeyPEM:  nodeKeyPEM,
	}, nil
}

// randomSerialNumber returns a random serial number suitable for x509 certificates.
func randomSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	sn, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}
	return sn, nil
}
