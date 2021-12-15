package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
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
		out     string
		genSeed bool
	)
	flag.StringVar(&out, "o", out, "output the keypair")
	flag.BoolVar(&genSeed, "seed", genSeed, "generate a seed instead of an ed25519 key pair")
	flag.Parse()

	if genSeed {
		seed := make([]byte, ed25519.SeedSize)
		if _, err := io.ReadFull(rand.Reader, seed); err != nil {
			return err
		}
		if len(out) > 0 {
			f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			if _, err = f.Write(seed); err != nil {
				return err
			}
			return nil
		}
		seedstr := hex.EncodeToString(seed)
		fmt.Println(seedstr)
		return nil
	}

	if len(out) == 0 {
		return errors.New("no output file")
	}

	keys, err := genKeyPair()
	if err != nil {
		return err
	}
	pems, err := openPemPair(out)
	if err == errFilesExist {
		fmt.Println(err.Error())
		return nil
	} else if err != nil {
		return err
	}
	defer pems.Close()

	err = keys.writeToPem(pems)
	if err != nil {
		return err
	}
	return nil

	// var (
	// 	pubFilename = fmt.Sprintf("%s.pub", out)
	// 	keyFilename = fmt.Sprintf("%s.key", out)
	// )

	// if exists(keyFilename) && exists(pubFilename) {
	// 	fmt.Println("private key file already exists")
	// 	return nil
	// }

	// pub, priv, err := ed25519.GenerateKey(rand.Reader)
	// if err != nil {
	// 	return err
	// }

	// privf, err := os.OpenFile(keyFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	// if err != nil {
	// 	return nil
	// }
	// defer privf.Close()
	// pubf, err := os.OpenFile(pubFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	// if err != nil {
	// 	return err
	// }
	// defer pubf.Close()

	// pkcs8Private, err := x509.MarshalPKCS8PrivateKey(priv)
	// if err != nil {
	// 	return err
	// }
	// pkixPublic, err := x509.MarshalPKIXPublicKey(pub)
	// if err != nil {
	// 	return err
	// }
	// if err = pem.Encode(privf, &pem.Block{
	// 	Type:    "PRIVATE KEY",
	// 	Headers: map[string]string{},
	// 	Bytes:   pkcs8Private,
	// }); err != nil {
	// 	return err
	// }
	// if err = pem.Encode(pubf, &pem.Block{
	// 	Type:    "PUBLIC KEY",
	// 	Headers: map[string]string{},
	// 	Bytes:   pkixPublic,
	// }); err != nil {
	// 	return err
	// }
	// return nil
}

func genKeyPair() (*pair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &pair{private: priv, public: pub}, nil
}

type pair struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
}

func (p *pair) writeToPem(pems *pemPair) error {
	pkcs8Private, err := x509.MarshalPKCS8PrivateKey(p.private)
	if err != nil {
		return err
	}
	pkixPublic, err := x509.MarshalPKIXPublicKey(p.public)
	if err != nil {
		return err
	}
	if err = pem.Encode(pems.private, &pem.Block{
		Type:    "PRIVATE KEY",
		Headers: map[string]string{},
		Bytes:   pkcs8Private,
	}); err != nil {
		return err
	}
	if err = pem.Encode(pems.public, &pem.Block{
		Type:    "PUBLIC KEY",
		Headers: map[string]string{},
		Bytes:   pkixPublic,
	}); err != nil {
		return err
	}
	return nil
}

func openPemPair(baseFilename string) (*pemPair, error) {
	var (
		err          error
		pems         pemPair
		pubFilename  = fmt.Sprintf("%s.pub", baseFilename)
		privFilename = fmt.Sprintf("%s.key", baseFilename)
	)
	if exists(pubFilename) && exists(privFilename) {
		return nil, errFilesExist
	}
	pems.private, err = createPemFile(privFilename)
	if err != nil {
		return nil, err
	}
	pems.public, err = createPemFile(pubFilename)
	if err != nil {
		pems.private.Close()
		return nil, err
	}
	return &pems, nil
}

type pemPair struct {
	private, public *os.File
}

var errFilesExist = errors.New("key pair already exists")

func (pp *pemPair) Close() error {
	var e1, e2 error
	if pp.private != nil {
		e1 = pp.private.Close()
	}
	if pp.public != nil {
		e2 = pp.public.Close()
	}
	if e1 != nil {
		return e1
	}
	return e2
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func createPemFile(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
}
