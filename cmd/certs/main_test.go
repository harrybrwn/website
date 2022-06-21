package main

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
)

func Test(t *testing.T) {
	ca, key, err := generatePair(nil, nil, pkix.Name{
		CommonName: "testing-root-ca",
	})
	if err != nil {
		t.Fatal(err)
	}
	cert, _, err := generatePair(ca, key, pkix.Name{
		CommonName: "testing-cert",
	})
	if err != nil {
		t.Fatal(err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(ca)
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: x509.NewCertPool(),
	}
	_, err = cert.Verify(opts)
	if err != nil {
		t.Fatal(err)
	}
}
