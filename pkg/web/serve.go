package web

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type FileServer struct {
	files map[string]*file

	// Custom error handling
	notFound func(file File, w io.Writer)
	failure  func(err error, file File, w io.Writer)

	preHooks []PreHook
	reqHooks []RequestHook
	resHooks []ResponseHook

	headers []header
}

type File interface {
	Info() fs.FileInfo
	Filepath() string
	ContentType() string
	ContentEncoding() string
}

type (
	PreHook      func(r *http.Request) error
	RequestHook  func(file File, r *http.Request) error
	ResponseHook func(file File, headers http.Header) error
)

type Hook struct {
	// Intercept the beginning of a request
	Pre PreHook
	// Intercept the incoming request for a file.
	Request RequestHook
	// Intercept the outgoing response to modify the response.
	Response ResponseHook
}

type FileServerOption func(*FileServer)

func WithHook(h Hook) FileServerOption {
	return func(fs *FileServer) {
		if h.Pre != nil {
			fs.preHooks = append(fs.preHooks, h.Pre)
		}
		if h.Request != nil {
			fs.reqHooks = append(fs.reqHooks, h.Request)
		}
		if h.Response != nil {
			fs.resHooks = append(fs.resHooks, h.Response)
		}
	}
}

func WithNotFoundHandler(fn func(file File, w io.Writer)) FileServerOption {
	return func(fs *FileServer) { fs.notFound = fn }
}

func WithFailureHook(fn func(err error, file File, w io.Writer)) FileServerOption {
	return func(fs *FileServer) { fs.failure = fn }
}

func WithURIHeaders(uri string, headers http.Header) FileServerOption {
	return func(fs *FileServer) { fs.ApplyHeaders(uri, headers) }
}

func WithGlobalHeaders(headers http.Header) FileServerOption {
	return func(fs *FileServer) { fs.ApplyGlobalHeaders(headers) }
}

func NewFileServer(fsys fs.FS, opts ...FileServerOption) (*FileServer, error) {
	files := make(map[string]*file)
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		var uri string
		if d.Name() == "index.html" {
			uri = filepath.Join("/", filepath.Dir(path))
		} else {
			uri = filepath.Join("/", path)
		}
		file, err := createFile(fsys, path, d)
		if err != nil {
			return err
		}
		files[uri] = file
		return nil
	})
	if err != nil {
		return nil, err
	}
	fs := &FileServer{files: files}
	for _, opt := range opts {
		opt(fs)
	}
	if fs.failure == nil {
		fs.failure = func(error, File, io.Writer) {}
	}
	if fs.notFound == nil {
		fs.notFound = func(File, io.Writer) {}
	}
	return fs, nil
}

func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := fs.pre(r)
	if err != nil {
		handleHookError(w, err)
		return
	}
	if r.Method != http.MethodGet {
		// This server is read-only!
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	file, ok := fs.files[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Apply request hooks
	if err = fs.onReq(file, r); err != nil {
		handleHookError(w, err)
		return
	}

	header := w.Header()
	if len(file.contentEncoding) > 0 {
		header.Set("Content-Encoding", file.contentEncoding)
	}
	if len(file.contentType) > 0 {
		header.Set("Content-Type", file.contentType)
	}
	header.Add("Content-Length", file.contentLength)
	for _, h := range fs.headers {
		header.Set(h.key, h.val)
	}
	for _, h := range file.headers {
		header.Set(h.key, h.val)
	}

	// Apply response hooks
	if err = fs.onRes(file, header); err != nil {
		handleHookError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(file.body)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"uri":  r.URL.Path,
			"file": file.fullname,
		}).Error("could not write to response body")
	}
}

func (fs *FileServer) Alias(origin, alias string) error {
	if _, ok := fs.files[alias]; ok {
		return errors.New("alias already exists")
	}
	file, ok := fs.files[origin]
	if !ok {
		return errors.New("static file not found")
	}
	fs.files[alias] = file
	return nil
}

func (fs *FileServer) ApplyHeaders(uri string, headers http.Header) error {
	file, ok := fs.files[uri]
	if !ok {
		return errors.New("file not found")
	}
	file.addHeaders(headers)
	return nil
}

func (fs *FileServer) ApplyGlobalHeaders(headers http.Header) {
	for key, val := range headers {
		fs.headers = append(fs.headers, header{
			key: key,
			val: val[0],
		})
	}
}

func (fs *FileServer) Files() []string {
	files := make([]string, 0, len(fs.files))
	for _, f := range fs.files {
		files = append(files, f.fullname)
	}
	return files
}

func (fs *FileServer) Routes() []string {
	routes := make([]string, 0, len(fs.files))
	for key := range fs.files {
		routes = append(routes, key)
	}
	return routes
}

func (fs *FileServer) pre(r *http.Request) (err error) {
	for _, h := range fs.preHooks {
		if err = h(r); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FileServer) onReq(file File, r *http.Request) (err error) {
	for _, h := range fs.reqHooks {
		if err = h(file, r); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FileServer) onRes(file File, headers http.Header) (err error) {
	for _, h := range fs.resHooks {
		if err = h(file, headers); err != nil {
			return err
		}
	}
	return nil
}

func handleHookError(rw http.ResponseWriter, err error) {
	switch e := err.(type) {
	case *Error:
		// TODO handle our own error type
		rw.WriteHeader(e.Status)
		logger.WithFields(logrus.Fields{
			"status":   e.Status,
			"code":     e.Code,
			"message":  e.Message,
			"internal": e.Internal,
		}).Error("failed to execute file server hook")
	default:
		rw.WriteHeader(http.StatusInternalServerError)
		logger.WithError(err).Error("failed to execute file server hook")
	}
}

type header struct{ key, val string }

type file struct {
	body            []byte
	fullname        string
	contentType     string
	contentEncoding string
	info            fs.FileInfo
	headers         []header
	contentLength   string
}

func (f *file) Info() fs.FileInfo       { return f.info }
func (f *file) Filepath() string        { return f.fullname }
func (f *file) ContentType() string     { return f.contentType }
func (f *file) ContentEncoding() string { return f.contentEncoding }

func (f *file) addHeaders(h http.Header) {
	for key, val := range h {
		f.headers = append(f.headers, header{
			key: key,
			val: val[0],
		})
	}
}

func createFile(fsys fs.FS, name string, d fs.DirEntry) (*file, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	file := file{
		fullname: name,
		body:     body,
	}
	file.contentType, file.contentEncoding, err = findMimeTypes(&file, name)
	if err != nil {
		return nil, err
	}

	info, err := d.Info()
	if err != nil {
		return nil, err
	}
	file.info = info
	file.contentLength = strconv.FormatInt(info.Size(), 10)
	return &file, nil
}

func findMimeTypes(file *file, fullname string) (string, string, error) {
	ct, ok := contentTypeFromExt(file, fullname)
	if ok {
		return ct, "", nil
	}
	ct = http.DetectContentType(file.body)
	switch ct {
	// TODO Do "compress" (lempel-ziv-welch), "deflate" (zlib/rfc1950), and "br" (brotli)
	case "application/x-gzip":
		ctype, err := gzippedContentType(file, fullname)
		if err != nil {
			return "", "", err
		}
		return ctype, "gzip", nil
	}
	return ct, "", nil
}

func contentTypeFromExt(file *file, fullname string) (string, bool) {
	ext := filepath.Ext(fullname)
	switch ext {
	case ".js":
		return "application/javascript", true
	case ".css":
		return "text/css", true
	case ".svg":
		return "image/svg+xml", true
	case ".json":
		var m interface{}
		err := json.Unmarshal(file.body, &m)
		if err != nil {
			return "", false
		}
		return "application/json", true
	default:
		return "", false
	}
}

func gzippedContentType(file *file, fullname string) (string, error) {
	buf := bytes.NewBuffer(file.body)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return "", err
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if len(fullname) > 0 && strings.Index(fullname, ".gz") == len(fullname)-3 {
		fullname = strings.Replace(fullname, ".gz", "", -1)
	}
	ct, ok := contentTypeFromExt(file, fullname)
	if ok {
		return ct, nil
	}
	return http.DetectContentType(body), nil
}
