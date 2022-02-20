package web

import (
	"fmt"
	"io"
	"io/fs"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/matryer/is"
)

func TestServePages(t *testing.T) {
	is := is.New(t)
	e := echo.New()
	type table struct {
		path  string
		ctype string
	}

	files := os.DirFS("testdata")

	// entries, err := fs.ReadDir(files, ".")
	// is.NoErr(err) // should be able to read directory
	// for _, e := range entries {
	// 	fmt.Println(e.Type(), e.Name())
	// }

	for _, tt := range []table{
		{path: "/admin", ctype: "text/html"},
		{path: "/", ctype: "text/html"},
		// {path: "/static/js/home.js", ctype: "text/javascript"},
	} {
		req := httptest.NewRequest("GET", tt.path, nil)
		req.Header.Set("content-type", tt.ctype)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := pageHandler(files)
		err := h(c)
		is.NoErr(err)
	}
}

func pageHandler(files fs.FS) func(c echo.Context) error {
	return func(c echo.Context) error {
		var (
			file  fs.File
			stat  fs.FileInfo
			err   error
			now   = time.Now()
			req   = c.Request()
			path  = req.URL.Path
			ctype = req.Header.Get("Content-Type")
		)

		switch {
		case strings.Contains(ctype, "text/html"):
			fallthrough
		case strings.Contains(ctype, "*/*"):
			path, err = filepath.Rel("/", path)
			if err != nil {
				return echo.NewHTTPError(500).SetInternal(err)
			}
			if path == "." {
				file, err = files.Open("index.html")
			} else {
				fmt.Println(path)
				file, err = files.Open(path + ".html")
				if os.IsNotExist(err) {
					fmt.Println(filepath.Join(path, "index.html"))
					file, err = files.Open(filepath.Join(path, "index.html"))
				}
			}
		default:
			file, err = files.Open(path)
		}

		if os.IsNotExist(err) {
			return echo.NewHTTPError(404).SetInternal(err)
		} else if err != nil {
			return err
		}
		defer file.Close()

		stat, err = file.Stat()
		if err != nil {
			return err
		}
		fmt.Println(time.Since(now))
		req.Header.Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
		c.Response().WriteHeader(200)
		_, err = io.Copy(c.Response(), file)
		return err
	}
}
