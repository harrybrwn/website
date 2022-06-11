package main

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/minio/madmin-go"
)

func Test(t *testing.T) {
	var p = madmin.PolicyInfo{
		PolicyName: "yee",
	}
	json.NewEncoder(os.Stdout).Encode(&p)
	c, err := madmin.New("", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(c)
}
