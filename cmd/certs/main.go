package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	mrand "math/rand"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	//caCertFile = "./config/pki/certs/ca.crt"
	//caKeyFile = "./config/pki/certs/ca.key"

	KeySize = 2048
)

func main() {
}

func NewCLI(cert, key string) (*Cli, error) {
	ca, caKey, err := loadPair(cert, key)
	if err != nil {
		return nil, err
	}
	return &Cli{ca: ca, key: caKey}, nil
}

type Cli struct {
	ca  *x509.Certificate
	key *rsa.PrivateKey
}

func NewGenerateCmd() *cobra.Command {
	c := cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
	}
	return &c
}

func exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

func loadPair(cert, key string) (*x509.Certificate, *rsa.PrivateKey, error) {
	kf, err := os.Open(key)
	if err != nil {
		return nil, nil, err
	}
	defer kf.Close()
	cf, err := os.Open(cert)
	if err != nil {
		return nil, nil, err
	}
	kb, err := io.ReadAll(kf)
	if err != nil {
		return nil, nil, err
	}
	cb, err := io.ReadAll(cf)
	if err != nil {
		return nil, nil, err
	}
	kblock, _ := pem.Decode(kb)
	cblock, _ := pem.Decode(cb)
	k, err := x509.ParsePKCS1PrivateKey(kblock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	c, err := x509.ParseCertificate(cblock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return c, k, nil
}

type templateOpt func(*x509.Certificate)

func generatePair(ca *x509.Certificate, caKey *rsa.PrivateKey, subject pkix.Name, opts ...templateOpt) (*x509.Certificate, *rsa.PrivateKey, error) {
	template := x509.Certificate{
		Version:            3,
		SerialNumber:       big.NewInt(mrand.Int63()),
		Subject:            subject,
		SignatureAlgorithm: x509.SHA256WithRSA,
		NotBefore:          time.Unix(time.Now().Unix()-60, 0),
		NotAfter:           time.Now().Add(time.Hour * 24 * 365),
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
			x509.ExtKeyUsageServerAuth,
		},
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	for _, o := range opts {
		o(&template)
	}

	key, err := rsa.GenerateKey(rand.Reader, KeySize)
	if err != nil {
		return nil, nil, err
	}
	if ca == nil && caKey == nil {
		template.IsCA = true
		template.BasicConstraintsValid = true
		caKey = key
		ca = &template
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, ca, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}
	c, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, err
	}
	return c, key, nil
}
