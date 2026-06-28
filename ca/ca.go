package ca

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type CA struct {
	key       *ecdsa.PrivateKey
	cert      *x509.Certificate
	certBytes []byte
	cache     map[string]*tls.Certificate
	mu        sync.Mutex
}

func Generate(dataDir string) (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "github-proxy CA",
		},
		NotBefore:             now,
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		MaxPathLen:            0,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	keyPath := filepath.Join(dataDir, "ca.key")
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	defer keyFile.Close()

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, err
	}

	certPath := filepath.Join(dataDir, "ca.crt")
	certFile, err := os.Create(certPath)
	if err != nil {
		return nil, err
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return nil, err
	}

	return &CA{
		key:       key,
		cert:      cert,
		certBytes: certBytes,
		cache:     make(map[string]*tls.Certificate),
	}, nil
}

func Load(dataDir string) (*CA, error) {
	keyPath := filepath.Join(dataDir, "ca.key")
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, os.ErrInvalid
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	certPath := filepath.Join(dataDir, "ca.crt")
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	certBlock, _ := pem.Decode(certData)
	if certBlock == nil {
		return nil, os.ErrInvalid
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return &CA{
		key:       key,
		cert:      cert,
		certBytes: certBlock.Bytes,
		cache:     make(map[string]*tls.Certificate),
	}, nil
}

func (c *CA) SignHost(host string) (*tls.Certificate, error) {
	c.mu.Lock()
	if cached, ok := c.cache[host]; ok {
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()

	hostKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore: now,
		NotAfter:  now.Add(825 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		DNSNames: []string{host},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, c.cert, &hostKey.PublicKey, c.key)
	if err != nil {
		return nil, err
	}

	tlsCert := &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  hostKey,
	}

	c.mu.Lock()
	c.cache[host] = tlsCert
	c.mu.Unlock()

	return tlsCert, nil
}

func randomSerial() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}
