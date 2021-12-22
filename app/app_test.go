package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"harrybrown.com/pkg/web"
)

func Test(t *testing.T) {
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
