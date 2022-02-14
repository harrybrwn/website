package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"syscall"

	"golang.org/x/term"
	"harrybrown.com/app"
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

func fromStdIn() ([]byte, error) {
	fmt.Print("Password: ")
	pw, err := term.ReadPassword(int(syscall.Stdin))
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
	return pw, nil
}
