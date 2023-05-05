package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"harrybrown.com/pkg/app/chat"
	"nhooyr.io/websocket"
)

func main() {
	var (
		ctx        = context.Background()
		port       = 8080
		host       = "localhost"
		room, user int
	)
	flag.IntVar(&port, "port", port, "chat server port")
	flag.StringVar(&host, "host", host, "chat server hostname")
	flag.IntVar(&room, "room", 0, "chat server room")
	flag.IntVar(&user, "user", 0, "user id")
	flag.Parse()

	conn, _, err := websocket.Dial(ctx, (&url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, strconv.Itoa(port)),
		Path:   filepath.Join("/api/chat", strconv.Itoa(room), "connect"),
		RawQuery: (url.Values{
			"user": {strconv.Itoa(user)},
		}).Encode(),
	}).String(), &websocket.DialOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(websocket.StatusGoingAway, "")
	err = run(ctx, &Client{
		conn: conn,
		room: room,
		user: user,
	})
	if err != nil {
		log.Fatal(err)
	}
}

type Client struct {
	conn       *websocket.Conn
	room, user int
}

func run(ctx context.Context, c *Client) error {
	var (
		lock sync.Mutex
		wg   sync.WaitGroup
		errs = make(chan error)
		sc   = bufio.NewScanner(os.Stdin)
	)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(2)
	go func() {
		wg.Wait()
		close(errs)
	}()

	go func() {
		defer wg.Done()
		for {
			_, r, err := c.conn.Reader(ctx)
			if errors.Is(err, io.EOF) {
				continue
			} else if err != nil {
				errs <- errors.Wrap(err, "can't get reader")
				return
			}
			lock.Lock()
			_, err = io.Copy(os.Stdout, r)
			if err != nil {
				errs <- errors.Wrap(err, "could not copy to stdout")
				lock.Unlock()
				return
			}
			_, err = os.Stdout.Write([]byte{'\n'})
			if err != nil {
				errs <- errors.Wrap(err, "could not write newline to stdout")
				lock.Unlock()
				return
			}
			lock.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		for {
			lock.Lock()
			fmt.Print("> ")
			lock.Unlock()
			if !sc.Scan() {
				break
			}
			line := sc.Text()
			if len(line) == 0 {
				log.Println("no message")
				continue
			}
			msg := chat.Message{
				Body:      line,
				Room:      c.room,
				UserID:    c.user,
				CreatedAt: time.Now(),
			}
			raw, err := json.Marshal(&msg)
			if err != nil {
				errs <- err
				return
			}
			w, err := c.conn.Writer(ctx, websocket.MessageText)
			if errors.Is(err, io.EOF) {
				log.Println(err)
				continue
			}
			if err != nil {
				errs <- err
				return
			}
			if _, err = w.Write(raw); err != nil {
				errs <- err
				return
			}
			if err = w.Close(); err != nil {
				errs <- err
				return
			}
		}
	}()
	return <-errs
}
