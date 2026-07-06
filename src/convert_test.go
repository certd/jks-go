package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/pavel-v-chernykh/keystore-go/v4"
	"software.sslmate.com/src/go-pkcs12"
)

func generateTestCertAndKey() (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "test.example.com",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

func createTestP12(cert *x509.Certificate, key *rsa.PrivateKey, password string) (string, error) {
	pfxData, err := pkcs12.Encode(rand.Reader, key, cert, nil, password)
	if err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp("", "test-*.p12")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(pfxData); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func createTestP12WithCA(cert *x509.Certificate, key *rsa.PrivateKey, caCerts []*x509.Certificate, password string) (string, error) {
	pfxData, err := pkcs12.Encode(rand.Reader, key, cert, caCerts, password)
	if err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp("", "test-ca-*.p12")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(pfxData); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func createTestPEMFiles(cert *x509.Certificate, key *rsa.PrivateKey) (certFile, keyFile string, err error) {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	certTmp, err := os.CreateTemp("", "test-cert-*.pem")
	if err != nil {
		return "", "", err
	}
	certTmp.Write(certPEM)
	certTmp.Close()

	keyTmp, err := os.CreateTemp("", "test-key-*.pem")
	if err != nil {
		os.Remove(certTmp.Name())
		return "", "", err
	}
	keyTmp.Write(keyPEM)
	keyTmp.Close()

	return certTmp.Name(), keyTmp.Name(), nil
}

func createTestCombinedPEM(cert *x509.Certificate, key *rsa.PrivateKey) (string, error) {
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	combined := append(certPEM, keyPEM...)

	tmpFile, err := os.CreateTemp("", "test-bundle-*.pem")
	if err != nil {
		return "", err
	}
	tmpFile.Write(combined)
	tmpFile.Close()

	return tmpFile.Name(), nil
}

func TestPKCS12ToJKS(t *testing.T) {
	cert, key, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	password := "testpassword"

	p12File, err := createTestP12(cert, key, password)
	if err != nil {
		t.Fatalf("failed to create test p12: %v", err)
	}
	defer os.Remove(p12File)

	jksFile, err := os.CreateTemp("", "test-*.jks")
	if err != nil {
		t.Fatalf("failed to create temp jks: %v", err)
	}
	jksPath := jksFile.Name()
	jksFile.Close()
	defer os.Remove(jksPath)

	err = convertPKCS12ToJKS(p12File, password, jksPath, password, "")
	if err != nil {
		t.Fatalf("convertPKCS12ToJKS failed: %v", err)
	}

	ks := keystore.New()
	f, err := os.Open(jksPath)
	if err != nil {
		t.Fatalf("failed to open jks: %v", err)
	}
	defer f.Close()

	if err := ks.Load(f, []byte(password)); err != nil {
		t.Fatalf("failed to load jks: %v", err)
	}

	aliases := ks.Aliases()
	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(aliases))
	}

	if aliases[0] != "test.example.com" {
		t.Fatalf("expected alias 'test.example.com', got '%s'", aliases[0])
	}

	if !ks.IsPrivateKeyEntry(aliases[0]) {
		t.Fatal("entry is not a private key entry")
	}

	_, err = ks.GetPrivateKeyEntry(aliases[0], []byte(password))
	if err != nil {
		t.Fatalf("failed to get private key entry: %v", err)
	}
}

func TestPEMToJKS_SeparateFiles(t *testing.T) {
	cert, key, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	certFile, keyFile, err := createTestPEMFiles(cert, key)
	if err != nil {
		t.Fatalf("failed to create test pem files: %v", err)
	}
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	jksFile, err := os.CreateTemp("", "test-*.jks")
	if err != nil {
		t.Fatalf("failed to create temp jks: %v", err)
	}
	jksPath := jksFile.Name()
	jksFile.Close()
	defer os.Remove(jksPath)

	password := "testpassword"

	err = convertPEMToJKS(certFile, keyFile, "", jksPath, password, "myalias")
	if err != nil {
		t.Fatalf("convertPEMToJKS failed: %v", err)
	}

	ks := keystore.New()
	f, err := os.Open(jksPath)
	if err != nil {
		t.Fatalf("failed to open jks: %v", err)
	}
	defer f.Close()

	if err := ks.Load(f, []byte(password)); err != nil {
		t.Fatalf("failed to load jks: %v", err)
	}

	aliases := ks.Aliases()
	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(aliases))
	}

	if aliases[0] != "myalias" {
		t.Fatalf("expected alias 'myalias', got '%s'", aliases[0])
	}
}

func TestPEMToJKS_CombinedFile(t *testing.T) {
	cert, key, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	bundleFile, err := createTestCombinedPEM(cert, key)
	if err != nil {
		t.Fatalf("failed to create combined pem: %v", err)
	}
	defer os.Remove(bundleFile)

	jksFile, err := os.CreateTemp("", "test-*.jks")
	if err != nil {
		t.Fatalf("failed to create temp jks: %v", err)
	}
	jksPath := jksFile.Name()
	jksFile.Close()
	defer os.Remove(jksPath)

	password := "testpassword"

	err = convertPEMToJKS(bundleFile, "", "", jksPath, password, "")
	if err != nil {
		t.Fatalf("convertPEMToJKS failed: %v", err)
	}

	ks := keystore.New()
	f, err := os.Open(jksPath)
	if err != nil {
		t.Fatalf("failed to open jks: %v", err)
	}
	defer f.Close()

	if err := ks.Load(f, []byte(password)); err != nil {
		t.Fatalf("failed to load jks: %v", err)
	}

	aliases := ks.Aliases()
	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(aliases))
	}

	if aliases[0] != "test.example.com" {
		t.Fatalf("expected alias 'test.example.com', got '%s'", aliases[0])
	}
}

func TestAutoAlias(t *testing.T) {
	cert, key, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	password := "testpassword"

	p12File, err := createTestP12(cert, key, password)
	if err != nil {
		t.Fatalf("failed to create test p12: %v", err)
	}
	defer os.Remove(p12File)

	jksFile, err := os.CreateTemp("", "test-*.jks")
	if err != nil {
		t.Fatalf("failed to create temp jks: %v", err)
	}
	jksPath := jksFile.Name()
	jksFile.Close()
	defer os.Remove(jksPath)

	err = convertPKCS12ToJKS(p12File, password, jksPath, password, "")
	if err != nil {
		t.Fatalf("convertPKCS12ToJKS failed: %v", err)
	}

	ks := keystore.New()
	f, err := os.Open(jksPath)
	if err != nil {
		t.Fatalf("failed to open jks: %v", err)
	}
	defer f.Close()

	if err := ks.Load(f, []byte(password)); err != nil {
		t.Fatalf("failed to load jks: %v", err)
	}

	aliases := ks.Aliases()
	if aliases[0] != "test.example.com" {
		t.Fatalf("expected alias extracted from CN 'test.example.com', got '%s'", aliases[0])
	}
}

func TestInvalidPassword(t *testing.T) {
	cert, key, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	password := "correctpassword"

	p12File, err := createTestP12(cert, key, password)
	if err != nil {
		t.Fatalf("failed to create test p12: %v", err)
	}
	defer os.Remove(p12File)

	jksFile, err := os.CreateTemp("", "test-*.jks")
	if err != nil {
		t.Fatalf("failed to create temp jks: %v", err)
	}
	jksPath := jksFile.Name()
	jksFile.Close()
	defer os.Remove(jksPath)

	err = convertPKCS12ToJKS(p12File, "wrongpassword", jksPath, password, "")
	if err == nil {
		t.Fatal("expected error for incorrect password, got nil")
	}
}

func TestMissingFile(t *testing.T) {
	err := convertPKCS12ToJKS("nonexistent.p12", "pass", "out.jks", "pass", "")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseCertificate(t *testing.T) {
	cert, _, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})

	parsed, err := parseCertificate(certPEM)
	if err != nil {
		t.Fatalf("parseCertificate failed: %v", err)
	}

	if parsed.Subject.CommonName != "test.example.com" {
		t.Fatalf("expected CN 'test.example.com', got '%s'", parsed.Subject.CommonName)
	}
}

func TestParsePrivateKey(t *testing.T) {
	_, key, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal key: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	parsed, err := parsePrivateKey(keyPEM, "")
	if err != nil {
		t.Fatalf("parsePrivateKey failed: %v", err)
	}

	if parsed == nil {
		t.Fatal("parsed key is nil")
	}
}

func TestPKCS12WithCAChain(t *testing.T) {
	cert, key, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate test cert: %v", err)
	}

	caCert, _, err := generateTestCertAndKey()
	if err != nil {
		t.Fatalf("failed to generate CA cert: %v", err)
	}

	password := "testpassword"

	p12File, err := createTestP12WithCA(cert, key, []*x509.Certificate{caCert}, password)
	if err != nil {
		t.Fatalf("failed to create test p12 with CA: %v", err)
	}
	defer os.Remove(p12File)

	jksFile, err := os.CreateTemp("", "test-ca-*.jks")
	if err != nil {
		t.Fatalf("failed to create temp jks: %v", err)
	}
	jksPath := jksFile.Name()
	jksFile.Close()
	defer os.Remove(jksPath)

	err = convertPKCS12ToJKS(p12File, password, jksPath, password, "")
	if err != nil {
		t.Fatalf("convertPKCS12ToJKS with CA chain failed: %v", err)
	}

	ks := keystore.New()
	f, err := os.Open(jksPath)
	if err != nil {
		t.Fatalf("failed to open jks: %v", err)
	}
	defer f.Close()

	if err := ks.Load(f, []byte(password)); err != nil {
		t.Fatalf("failed to load jks: %v", err)
	}

	entry, err := ks.GetPrivateKeyEntry("test.example.com", []byte(password))
	if err != nil {
		t.Fatalf("failed to get private key entry: %v", err)
	}

	chain := entry.CertificateChain
	if len(chain) < 2 {
		t.Fatalf("expected at least 2 certs in chain (leaf + CA), got %d", len(chain))
	}
}
