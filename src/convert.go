package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pavel-v-chernykh/keystore-go/v4"
	"software.sslmate.com/src/go-pkcs12"
)

func convertPKCS12ToJKS(srcFile, srcPass, dstFile, dstPass, alias string) error {
	pfxData, err := os.ReadFile(srcFile)
	if err != nil {
		return fmt.Errorf("failed to read PKCS12 file: %w", err)
	}

	privateKey, cert, caCerts, err := pkcs12.DecodeChain(pfxData, srcPass)
	if err != nil {
		if errors.Is(err, pkcs12.ErrIncorrectPassword) || errors.Is(err, pkcs12.ErrDecryption) {
			return fmt.Errorf("incorrect password for PKCS12 file")
		}
		return fmt.Errorf("failed to decode PKCS12: %w", err)
	}

	return writeJKS(privateKey, cert, caCerts, dstFile, dstPass, alias)
}

func convertPEMToJKS(certFile, keyFile, srcPass, dstFile, dstPass, alias string) error {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate PEM file: %w", err)
	}

	cert, err := parseCertificate(certPEM)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	var keyData []byte
	if keyFile != "" {
		keyData, err = os.ReadFile(keyFile)
	} else {
		keyData = certPEM
	}
	if err != nil {
		return fmt.Errorf("failed to read private key PEM file: %w", err)
	}

	privateKey, err := parsePrivateKey(keyData, srcPass)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	return writeJKS(privateKey, cert, nil, dstFile, dstPass, alias)
}

func parseCertificate(data []byte) (*x509.Certificate, error) {
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			return cert, nil
		}
		data = rest
	}
	return nil, errors.New("no certificate found in PEM data")
}

func parsePrivateKey(data []byte, password string) (interface{}, error) {
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}

		var keyBytes []byte
		if x509.IsEncryptedPEMBlock(block) {
			if password == "" {
				return nil, errors.New("private key is encrypted but no password provided")
			}
			der, err := x509.DecryptPEMBlock(block, []byte(password))
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt private key: %w", err)
			}
			keyBytes = der
		} else {
			keyBytes = block.Bytes
		}

		var key interface{}
		var err error

		switch block.Type {
		case "PRIVATE KEY":
			key, err = x509.ParsePKCS8PrivateKey(keyBytes)
		case "RSA PRIVATE KEY":
			key, err = x509.ParsePKCS1PrivateKey(keyBytes)
		case "EC PRIVATE KEY":
			key, err = x509.ParseECPrivateKey(keyBytes)
		default:
			data = rest
			continue
		}

		if err != nil {
			return nil, err
		}
		return key, nil
	}
	return nil, errors.New("no private key found in PEM data")
}

func writeJKS(privateKey interface{}, cert *x509.Certificate, caCerts []*x509.Certificate, dstFile, dstPass, alias string) error {
	if alias == "" {
		alias = cert.Subject.CommonName
	}
	if alias == "" {
		alias = "certificate"
	}

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key to PKCS8: %w", err)
	}

	chain := []keystore.Certificate{
		{
			Type:    "X.509",
			Content: cert.Raw,
		},
	}
	for _, ca := range caCerts {
		chain = append(chain, keystore.Certificate{
			Type:    "X.509",
			Content: ca.Raw,
		})
	}

	entry := keystore.PrivateKeyEntry{
		CreationTime:     time.Now(),
		PrivateKey:       pkcs8Key,
		CertificateChain: chain,
	}

	ks := keystore.New()
	if err := ks.SetPrivateKeyEntry(alias, entry, []byte(dstPass)); err != nil {
		return fmt.Errorf("failed to set private key entry: %w", err)
	}

	f, err := os.Create(dstFile)
	if err != nil {
		return fmt.Errorf("failed to create JKS file: %w", err)
	}
	defer f.Close()

	if err := ks.Store(f, []byte(dstPass)); err != nil {
		return fmt.Errorf("failed to write JKS: %w", err)
	}

	return nil
}
