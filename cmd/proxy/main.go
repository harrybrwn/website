package main

import (
	"context"
	"io"
	"net"
	"os"
	"sync"

	"harrybrown.com/pkg/log"
)

var logger = log.GetLogger()

func main() {
	l, err := net.Listen("tcp", ":8086")
	if err != nil {
		logger.Fatal(err)
	}
	logger.WithField("addr", l.Addr()).Info("proxy started")
	run(context.Background(), l)
}

func run(ctx context.Context, l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.WithError(err).Error("failed to accept connection")
			continue
		}
		logger.WithFields(log.Fields{
			"remote-addr": conn.RemoteAddr().String(),
			"local-addr":  conn.LocalAddr().String(),
		}).Info("accepted connection")
		go func(c net.Conn) {
			defer c.Close()
			err = handle(ctx, "geoip:8084", c)
			if err != nil {
				logger.WithError(err).Error("failed to handle connection")
			}
		}(conn)
	}
}

func handle(ctx context.Context, addr string, c net.Conn) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	logger.WithFields(log.Fields{
		"host": host,
		"port": port,
	}).Info("addr parsed")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		n, err := io.Copy(os.Stdout, c)
		logger.WithFields(log.Fields{"n": n, "err": err}).Info("done reading")
	}()
	wg.Wait()
	return nil
}

func copy(w io.Writer, r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	num := 0
	for {
		n, err := r.Read(buf)
		num += n
		if err != nil {
			return num, err
		}
		n, err = w.Write(buf[:n])
		if err != nil {
			return num, err
		}
	}
}
