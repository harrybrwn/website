package web

import (
	"embed"
	_ "embed"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

var (
	//go:embed testdata
	testdata embed.FS
	//go:embed testdata/favicon.ico
	favicon string
	//go:embed testdata/sitemap.xml.gz
	sitemap string
	//go:embed testdata/app/app.js.gz
	appJsGz []byte
)

func init() { logger.SetOutput(io.Discard) }

func testFS(t *testing.T, opts ...FileServerOption) *FileServer {
	data, err := fs.Sub(testdata, "testdata")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := NewFileServer(data, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func TestFileNewServer(t *testing.T) {
	files := os.DirFS("./testdata")
	fserve, err := NewFileServer(files)
	if err != nil {
		t.Fatal(err)
	}
	if fserve.files == nil {
		t.Fatal("did not create files map")
	}
	for uri, file := range fserve.files {
		if len(uri) == 0 {
			t.Fatal("got zero length uri/key")
		}
		if uri[0] != '/' {
			t.Fatalf("uri does not start with forward slash: %q", uri)
		}
		if file.info == nil {
			t.Fatal("file has no info type")
		}
	}
}

func TestFileServerServeHTTP(t *testing.T) {
	type table struct {
		// input
		path string
		// expected output
		status int
		body   string
		ct     string // expected content-type
		ce     string // expected content-encoding
	}
	fs := testFS(t)
	for _, tt := range []table{
		{path: "/", status: 200, body: "<p>/index.html</p>", ct: "text/html"},
		{path: "/app/app.js", status: 200, body: "console.log(\"hello\");\n", ct: "application/javascript"},
		{path: "/app/data.json", status: 200, body: "{\"key\":\"value\",\"number\":10}", ct: "application/json"},
		{path: "/app", status: 200, body: `<!DOCTYPE html>
<html lang="en">
<head>
	<title>App</title>
</head>
<body>
	<script src="./app.js"></script>
</body>
</html>`, ct: "text/html"},
		{path: "/blog", status: 200, body: "<p>/blog/index.html</p>", ct: "text/html"},
		{path: "/not/a/file", status: 404},
		{path: "/favicon.ico", status: 200, body: favicon, ct: "image/x-icon"},
		{path: "/sitemap.xml.gz", status: 200, body: sitemap, ct: "text/xml", ce: "gzip"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", tt.path, nil)
		fs.ServeHTTP(rec, req)
		body := rec.Body.String()
		ct := rec.HeaderMap.Get("Content-Type")
		ce := rec.HeaderMap.Get("Content-Encoding")
		if rec.Code != tt.status {
			t.Errorf("wrong status code: got %d, want %d", rec.Code, tt.status)
			continue
		}
		if rec.Code >= 300 {
			// the next tests will fail for non-ok requests
			continue
		}
		cl, err := strconv.Atoi(rec.HeaderMap.Get("Content-Length"))
		if err != nil {
			t.Errorf("%q: could not parse Content-Length: %v", tt.path, err)
		}
		if body != tt.body {
			t.Errorf("%q: wrong body: got %q, want %q", tt.path, body, tt.body)
		}
		if len(body) != cl {
			t.Errorf("%q: content-length mismatch: got %d, want %d", tt.path, cl, len(body))
		}
		if !strings.HasPrefix(ct, tt.ct) {
			t.Errorf("%q: wrong content-type: got %q, want prefix of %q", tt.path, ct, tt.ct)
		}
		if len(tt.ce) > 0 {
			if ce != tt.ce {
				t.Errorf("%q: wrong content-encoding: got %q, want %q", tt.path, ce, tt.ce)
			}
		}
	}
}

func TestFileServerServeHTTP_Err(t *testing.T) {
	type table struct {
		method string
		status int
	}
	fs := testFS(t)
	for _, tt := range []table{
		{method: http.MethodHead, status: 405},
		{method: http.MethodPost, status: 405},
		{method: http.MethodPut, status: 405},
		{method: http.MethodPatch, status: 405},
		{method: http.MethodDelete, status: 405},
		{method: http.MethodConnect, status: 405},
		{method: http.MethodOptions, status: 405},
		{method: http.MethodTrace, status: 405},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, "/", nil)
		fs.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Errorf("wrong status: got %d, want %d", rec.Code, tt.status)
		}
	}
}

func TestFileServerHooks(t *testing.T) {
	var (
		didPre, didReq, didRes bool
	)
	fs := testFS(t, WithHook(Hook{
		Pre: func(r *http.Request) error {
			didPre = true
			return nil
		},
		Request: func(file File, r *http.Request) error {
			didReq = true
			return nil
		},
		Response: func(file File, headers http.Header) error {
			didRes = true
			headers.Set("Sec-CH-UA", "i don't know what this header is for")
			return nil
		},
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	fs.ServeHTTP(rec, req)
	if !didPre {
		t.Error("did not execute pre hook")
	}
	if !didReq {
		t.Error("did not execute request hook")
	}
	if !didRes {
		t.Error("did not execute response hook")
	}
	ua := rec.HeaderMap.Get("Sec-CH-UA")
	if ua != "i don't know what this header is for" {
		t.Error("header not set")
	}
}

func TestFindMimeType(t *testing.T) {
	f := file{body: []byte(sitemap)}
	contentType, contentEncoding, err := findMimeTypes(&f, "sitemap.xml.gz")
	if err != nil {
		t.Fatal(err)
	}
	expectedCt := "text/xml; charset=utf-8"
	expectedEt := "gzip"
	if contentType != expectedCt {
		t.Errorf("wrong content type: got %q, want %q", contentType, expectedCt)
	}
	if contentEncoding != expectedEt {
		t.Errorf("wrong content encoding: got %q, want %q", contentEncoding, expectedEt)
	}

	contentType, contentEncoding, err = findMimeTypes(&file{body: appJsGz}, "app.js.gz")
	if err != nil {
		t.Fatal(err)
	}
	expectedCt = "application/javascript"
	expectedEt = "gzip"
	if contentType != expectedCt {
		t.Errorf("wrong content type: got %q, want %q", contentType, expectedCt)
	}
	if contentEncoding != expectedEt {
		t.Errorf("wrong content encoding: got %q, want %q", contentEncoding, expectedEt)
	}
}

func TestContentTypeFromExt(t *testing.T) {
	for _, tt := range []struct {
		name, ct, body string
		ok             bool
	}{
		{"file.js", "application/javascript", "", true},
		{"file.css", "text/css", "", true},
		{"stuff.json", "application/json", `{"one":1,"two":"2","three":3.14159}`, true},
		{"/path/to/not-me.html", "", "", false},
		{"file.svg", "image/svg+xml", "", true},
	} {
		ct, ok := contentTypeFromExt(&file{body: []byte(tt.body)}, tt.name)
		if ok != tt.ok {
			t.Errorf("unexpected failure: contentTypeFromExt should have returned %v and did not", tt.ok)
			continue
		}
		if ct != tt.ct {
			t.Errorf("expected %q to have content type of %q", tt.name, tt.ct)
		}
	}
}
