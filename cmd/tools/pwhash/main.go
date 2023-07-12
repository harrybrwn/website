package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"

	"golang.org/x/term"
	"gopkg.hrry.dev/homelab/pkg/app"
)

func main() {
	var (
		pass    string
		pwBytes []byte
		err     error

		postgres bool
		hex      bool
	)
	flag.StringVar(&pass, "p", pass, "password to hash")
	flag.BoolVar(&postgres, "postgres", postgres, "output encoded as postgres-bytea safe")
	flag.BoolVar(&hex, "hex", hex, "output as hex")
	flag.Parse()

	if len(pass) == 0 {
		pwBytes, err = fromStdIn()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		pwBytes = []byte(pass)
	}
	hash, err := app.HashPassword(pwBytes)
	if err != nil {
		log.Fatal(err)
	}

	if postgres {
		fmt.Printf("postgres encoded: E'\\\\x%x'\n", hash)
	} else if hex {
		fmt.Printf("hex:              %x\n", hash)
	} else {
		fmt.Printf("raw:              %s\n", hash)
	}
}

func fromStdIn() (pw []byte, err error) {
	if flag.Arg(0) == "-" {
		var buf bytes.Buffer
		_, err = io.Copy(&buf, os.Stdin)
		if err != nil {
			return nil, err
		}
		pw = bytes.Trim(buf.Bytes(), "\n\t ")
	} else {
		fmt.Print("Password: ")
		pw, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return nil, err
		}
		fmt.Print("\nConfirm Password: ")
		confirmPw, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(confirmPw, pw) {
			return nil, errors.New("passwords were different")
		}
	}
	return pw, nil
}
