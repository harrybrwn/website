package urifs

import (
	"net/url"
	"testing"

	"github.com/matryer/is"
)

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
