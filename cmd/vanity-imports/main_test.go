package main

import (
	"bytes"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/matryer/is"
)

func init() {
	logger.SetOutput(io.Discard)
}

func TestGetPackageName(t *testing.T) {
	is := is.New(t)
	for _, tt := range []struct {
		path string
		err  error
		exp  string
	}{
		// happy path
		{path: "/package", exp: "package"},
		{path: "/package/name", exp: "package/name"},
		// errors
		{path: "/", err: errNoPackage},
		{path: "", err: errInvalidPath},
		{path: "package", err: errInvalidPath},
	} {
		req := httptest.NewRequest("GET", "/", nil)
		req.URL.Path = tt.path
		p, err := packageName(req)
		is.True(errors.Is(err, tt.err))
		is.Equal(p, tt.exp)
	}
}

func TestVanityImport(t *testing.T) {
	is := is.New(t)
	h := VanityImport(&Vanity{
		RepoURL: "https://github.com/torvalds",
		Domain:  "gopkg.example.com",
		Package: &Package{},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/package-name?go-get=1", nil)
	h(rec, req)
	res := rec.Result()
	is.Equal(res.StatusCode, 200)
	b, err := io.ReadAll(res.Body)
	is.NoErr(err)
	is.True(bytes.Contains(b, []byte("github.com/torvalds/package-name")))
}
