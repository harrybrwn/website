package app

import (
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"harrybrown.com/pkg/log"
)

const (
	// Name is the name of the application.
	Name               = "harrybrown.com"
	BuildDir           = "./build"
	StaticCacheControl = "public, max-age=31919000"
)

var (
	StartTime = time.Now()
)

func Page(raw []byte, filename string) echo.HandlerFunc {
	var (
		hf echo.HandlerFunc
	)
	filename = filepath.Join(BuildDir, filename)
	if Debug {
		hf = func(c echo.Context) error {
			return ServeFile(c, 200, filename)
		}
		b, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		if http.DetectContentType(b) == "application/x-gzip" {
			hf = asGzip(hf)
		}
	} else {
		ct := http.DetectContentType(raw)
		hf = func(c echo.Context) error {
			h := c.Response().Header()
			staticLastModified(h)
			h.Set("Cache-Control", StaticCacheControl)
			return c.Blob(200, ct, raw)
		}
		if http.DetectContentType(raw) == "application/x-gzip" {
			hf = asGzip(hf)
		}
	}
	return hf
}

func ServeFile(c echo.Context, status int, filename string) error {
	http.ServeFile(c.Response(), c.Request(), filename)
	return nil
}

func staticLastModified(h http.Header) {
	h.Set("Last-Modified", StartTime.UTC().Format(http.TimeFormat))
}

func asGzip(handler echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !acceptsGzip(c.Request().Header) {
			logger.WithField(
				"accept-encoding", c.Request().Header.Get("Accept-Encoding"),
			).Error("browser encoding not supported")
			return c.Blob(500, "text/html", []byte("<h2>encoding failure</h2>"))
		}
		c.Response().Header().Set("Content-Encoding", "gzip")
		return handler(c)
	}
}

func acceptsGzip(header http.Header) bool {
	accept := header.Get("Accept-Encoding")
	return strings.Contains(accept, "gzip")
}

// NewLogger creates a new logger that will intercept a handler and replace it
// with one that has logging functionality.
func NewLogger(h http.Handler) http.Handler {
	return &pageLogger{
		wrapped: h,
		l:       log.NewPlainLogger(os.Stdout),
	}
}

type pageLogger struct {
	wrapped http.Handler
	l       log.PrintLogger
}

func (p *pageLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.l.Printf("%s %s%s\n", r.Method, r.Host, r.URL)
	p.wrapped.ServeHTTP(w, r)
}

func NotFoundHandler(fs fs.FS) http.Handler {
	t, err := template.ParseFS(fs, "*/index.html", "*/pages/404.html")
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
		err = t.ExecuteTemplate(rw, "base", nil)
		if err != nil {
			log.Println(err)
		}
	})
}

// NotFound handles requests that generate a 404 error
func NotFound(w http.ResponseWriter, r *http.Request) {
	var tmplNotFound = template.Must(template.ParseFiles(
		"templates/pages/404.html",
		"templates/index.html",
	))
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if err := tmplNotFound.ExecuteTemplate(w, "base", nil); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}

// ParseFlags parses flags.
func ParseFlags() {
	defer recoverFlagHelpErr()
	flag.Parse()
}

// StringFlag adds a string flag with a shorthand.
func StringFlag(ptr *string, name, desc string) {
	flag.StringVar(ptr, name, *ptr, desc)
	flag.StringVar(ptr, name[:1], *ptr, desc)
}

// BoolFlag adds a boolean flag with a shorthand.
func BoolFlag(ptr *bool, name, desc string) {
	flag.BoolVar(ptr, name, *ptr, desc)
	flag.BoolVar(ptr, name[:1], *ptr, desc)
}

// RecoverFlagHelpErr will gracfully end the program if the help flag is given.
func recoverFlagHelpErr() {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			return
		}
		if err == flag.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Println("Error:", err.Error())
			os.Exit(2)
		}
	}
}

type appFlag struct {
	flag      *flag.Flag
	name      string
	shorthand string
	usage     string
	fmtlen    int
}

func (af *appFlag) len() int {
	return len(af.name)
}

func (af *appFlag) useline() string {
	if len(af.shorthand) != 0 {
		return fmt.Sprintf("-%s, -%s", af.shorthand, af.name)
	}
	return fmt.Sprintf("    -%s", af.name)
}

func (af *appFlag) defval() string {
	deflt := ""
	if af.flag != nil {
		deflt = fmt.Sprintf("(default: %s)", af.flag.DefValue)
	}
	return deflt
}

func (af *appFlag) String() string {
	return fmt.Sprintf("  %s%s%s %s",
		af.useline(),
		strings.Repeat(" ", 4+af.fmtlen-af.len()),
		af.usage,
		af.defval())
}

func newflag(flg *flag.Flag, shorthand string) *appFlag {
	return &appFlag{
		flag:      flg,
		name:      flg.Name,
		usage:     flg.Usage,
		shorthand: shorthand,
	}
}

func getFlags() ([]*appFlag, int) {
	fmap := make(map[string]*appFlag)
	flag.VisitAll(func(flg *flag.Flag) {
		if _, inmap := fmap[flg.Usage]; !inmap {
			fmap[flg.Usage] = new(appFlag)
			if len(flg.Name) == 1 {
				fmap[flg.Usage].shorthand = flg.Name
			}
		}
		fmap[flg.Usage] = newflag(flg, fmap[flg.Usage].shorthand)
	})
	length := len(fmap)

	flgs := make([]*appFlag, length)
	i := 0
	for _, fl := range fmap {
		flgs[i] = fl
		i++
	}

	max := flgs[0].len()
	for i = 1; i < length; i++ {
		if flgs[i].len() > max {
			max = flgs[i].len()
		}
	}

	sort.Slice(flgs, func(i, j int) bool {
		return strings.Compare(flgs[i].name, flgs[j].name) == 0
	})
	return flgs, max
}

var helpFlag = &appFlag{name: "help", shorthand: "h", usage: "get help", fmtlen: 4}

func init() {
	out := flag.CommandLine.Output()

	flag.CommandLine.Usage = func() {
		flags, maxlen := getFlags()
		helpFlag.fmtlen = maxlen
		flags = append(flags, helpFlag)

		fmt.Fprintf(out, "Usage of %s:\n", Name)
		for _, v := range flags {
			v.fmtlen = maxlen
			fmt.Fprintln(out, v)
		}
		fmt.Fprint(out, "\n")
	}
}
