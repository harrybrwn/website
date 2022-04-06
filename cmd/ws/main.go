package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"

	"github.com/pkg/errors"
	"nhooyr.io/websocket"
)

func main() {
	var path string
	flag.StringVar(&path, "p", path, "url path of websocket")
	flag.Parse()

	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   path,
	}
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "ending program")

	fmt.Println("connected:")
	messages := scanner(ctx)
	go func() {
		err = loop(ctx, conn, messages)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("loop error: %v\n", err)
		}
	}()
	<-ctx.Done()
}

func loop(ctx context.Context, conn *websocket.Conn, messages <-chan string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for blob := range messages {
		if len(blob) == 0 {
			fmt.Println("Warn: no message")
			continue
		}
		errs := make(chan error)
		done := make(chan struct{}, 2)

		go func() {
			defer func() { done <- struct{}{} }()
			w, err := conn.Writer(ctx, websocket.MessageText)
			if err != nil {
				// return err
				errs <- err
				return
			}
			defer func() {
				if err = w.Close(); err != nil {
					// return errors.Wrap(err, "could not close writer")
					errs <- errors.Wrap(err, "could not close writer")
					return
				}
			}()
			_, err = w.Write([]byte(blob))
			if err != nil {
				// return errors.Wrap(err, "could not write blob")
				errs <- errors.Wrap(err, "could not write blob")
				return
			}
		}()

		go func() {
			defer func() { done <- struct{}{} }()
			_, r, err := conn.Reader(ctx)
			if err != nil {
				// return errors.Wrap(err, "could not get reader")
				errs <- errors.Wrap(err, "could not get reader")
				return
			}
			_, err = io.Copy(os.Stdout, r)
			if err != nil {
				// return errors.Wrap(err, "could not read from connection")
				errs <- errors.Wrap(err, "could not read from connection")
				return
			}
		}()

		println()
		if err := <-errs; err != nil {
			return err
		}
		select {
		case err := <-errs:
			if err != nil {
				return err
			}
		case <-done:
		}
	}
	return nil
}

func scanner(ctx context.Context) <-chan string {
	ch := make(chan string)
	sc := bufio.NewScanner(os.Stdin)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if !sc.Scan() {
				return
			}
			blob := sc.Text()
			fmt.Println("scanned blob:", blob)
			select {
			case <-ctx.Done():
				return
			case ch <- blob:
			}
		}
	}()
	return ch
}
