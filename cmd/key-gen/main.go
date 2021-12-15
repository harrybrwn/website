package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		out string
	)
	flag.StringVar(&out, "o", out, "output the keypair")
	flag.Parse()

	if len(out) == 0 {
		return errors.New("no output file")
	}

	var (
		pubFilename = fmt.Sprintf("%s.pub", out)
		keyFilename = fmt.Sprintf("%s.key", out)
	)

	if exists(keyFilename) && exists(pubFilename) {
		fmt.Println("private key file already exists")
		return nil
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	privf, err := os.OpenFile(keyFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil
	}
	defer privf.Close()
	pubf, err := os.OpenFile(pubFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer pubf.Close()

	pkcs8Private, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}
	pkixPublic, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return err
	}
	if err = pem.Encode(privf, &pem.Block{
		Type:    "PRIVATE KEY",
		Headers: map[string]string{},
		Bytes:   pkcs8Private,
	}); err != nil {
		return err
	}
	if err = pem.Encode(pubf, &pem.Block{
		Type:    "PUBLIC KEY",
		Headers: map[string]string{},
		Bytes:   pkixPublic,
	}); err != nil {
		return err
	}
	return nil
}

func stop(e error) {
	if e != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		os.Exit(1)
	}
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
