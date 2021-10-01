package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"harrybrown.com/pkg/web"
)

func TestFindUrl(t *testing.T) {
	u, err := url.Parse("http://harrybrwn.com")
	if err != nil {
		t.Error(err)
	}

	var img string
	img = findImage(u, "lg")
	if img != "2250x3000" {
		t.Error("got the wrong image folder from findImage")
	}

	u, err = url.Parse("http://harrybrwn.com/static/img.jpg")
	img = findImage(u, "xs")
	if img != "/static/563x750/img.jpg" {
		t.Error("bad result from findImage:", img)
	}
}

func init() {
	cwd, _ := os.Getwd()
	dir := filepath.Base(cwd)
	if dir == "app" {
		os.Chdir("..")
	}
}

func testGetReq(r web.Route, t *testing.T) {
	var (
		err   error
		nodes []web.Route
	)
	if nodes, err = r.Expand(); nodes != nil {
		if err != nil {
			t.Errorf("got error expanding %s: %s\n", r.Path(), err.Error())
		}
		for _, node := range nodes {
			testGetReq(node, t)
		}
	}

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", r.Path(), nil)
	r.Handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Error("bad response code from", r.Path(), "got", rr.Code)
	}
}

func TestFileServer(t *testing.T) {
	var (
		rr  = httptest.NewRecorder()
		req *http.Request
	)
	fs := web.NewRoute("/static/", NewFileServer("static"))
	req, _ = http.NewRequest("GET", "/static/img/github.svg", nil)

	fs.Handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Error("bad responce code from file server")
	}
	if len(rr.Body.Bytes()) < 1 {
		t.Error("the file server did not get anything")
	}

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/static/filenothere.txt", nil)
	fs.Handler().ServeHTTP(rr, req)
	if rr.Code == 200 {
		t.Error("file not found should not give a successful get request")
	}
	if rr.Code != 404 {
		t.Error("why is the code not 404!!")
	}

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/static/img/me.jpg?size=sm", nil)
	fs.Handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Error("bad response code")
	}

	rr = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/static/img/me.jpg?size=md", nil)
	fs.Handler().ServeHTTP(rr, req)

	if req.URL.Path != "/static/img/1688x2251/me.jpg" {
		t.Error("wrong url")
	}
	if rr.Code != 200 {
		t.Error("bad response code")
	}
}
