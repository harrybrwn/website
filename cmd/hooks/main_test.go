package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/signer"
)

func TestVerifyHookSignature(t *testing.T) {
}

// Signature and API related constants.
const (
	//signV4Algorithm   = "AWS4-HMAC-SHA256"
	iso8601DateFormat = "20060102T150405Z"
	//yyyymmdd          = "20060102"
)

func TestS3V4Signature(t *testing.T) {
	fmt.Println(signer.SignV4(http.Request{}, "", "", "", ""))
	date, err := time.Parse(iso8601DateFormat, "20220623T062957Z")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(date)
}
