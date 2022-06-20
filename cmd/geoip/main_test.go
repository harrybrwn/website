package main

import (
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/matryer/is"
)

func TestGetIP(t *testing.T) {
	is := is.New(t)
	ips := parseIPList("73.74.75.76, 73.72.71.70")
	is.Equal(len(ips), 2)
	is.Equal(ips[0], net.IPv4(73, 74, 75, 76))
	is.Equal(ips[1], net.IPv4(73, 72, 71, 70))
	ips = parseIPList("1.1.1.1")
	is.Equal(len(ips), 1)
	is.Equal(ips[0], net.IPv4(1, 1, 1, 1))
	r := &http.Request{Header: http.Header{
		"X-Forwarded-For": {"73.74.75.76", "73.72.71.70"},
	}}
	ip, err := getIP(r)
	is.NoErr(err)
	is.Equal(ip, net.IPv4(73, 74, 75, 76))
	r = &http.Request{RemoteAddr: "8.8.8.8:9999"}
	ip, err = getIP(r)
	is.NoErr(err)
	is.Equal(ip, net.IPv4(8, 8, 8, 8))
}

func TestObjectRequestFromURI(t *testing.T) {
	is := is.New(t)
	u, err := url.Parse("s3://admin@password:localhost:9000/the-bucket/path/to/object")
	if err != nil {
		t.Fatal(err)
	}
	in := objectRequestFromURI(u)
	is.Equal(*in.Bucket, "the-bucket")
	is.Equal(*in.Key, "path/to/object")
}
