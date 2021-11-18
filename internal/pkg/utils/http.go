package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/cli"
)

// GetTLSCerts get cert file path, return (certPath, keyPath, error)
func GetTLSCerts() (string, string, error) {
	// FIXME: hardcode
	certDir := "certs"
	certDir, err := filepath.Abs(certDir)
	if err != nil {
		logger.Errorf("get absolute path failed: %s", err)
		return "", "", err
	}
	if err := EnsureDir(certDir); err != nil {
		return "", "", err
	}

	certPath := fmt.Sprintf("%s/server.crt", certDir)
	keyPath := fmt.Sprintf("%s/server.key", certDir)

	return certPath, keyPath, nil
}

// NewTLSCert check or create TLS cert, return (certPath, keyPath, error)
func NewTLSCert() (string, string, error) {
	certPath, keyPath, err := GetTLSCerts()
	if err != nil {
		return "", "", nil
	}
	if FileExist(certPath) && FileExist(keyPath) {
		return certPath, keyPath, nil
	}

	logger.Infof("create cert: %s key: %s ...", certPath, keyPath)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	ipAddrs := []net.IP{net.ParseIP("127.0.0.1")}
	for _, ip := range cli.GetConfig().SSLCertIPAddresses {
		ipAddrs = append(ipAddrs, ip)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
			CommonName:   "*",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           ipAddrs,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	pk, _ := rsa.GenerateKey(rand.Reader, 2048)

	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &pk.PublicKey, pk)
	certOut, _ := os.Create(certPath)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, _ := os.Create(keyPath)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)})
	keyOut.Close()

	return certPath, keyPath, nil
}

// NewHTTPClient return *http.Client with `cacert` config
func NewHTTPClient() (*http.Client, error) {
	certPath, keyPath, err := NewTLSCert()
	if err != nil {
		return nil, err
	}

	caCert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: true}
	client := &http.Client{Timeout: 30 * time.Second, Transport: transport}

	return client, nil
}
