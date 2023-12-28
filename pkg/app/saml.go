package app

import (
	"crypto/rsa"
	"net/http"
	"net/url"

	"github.com/crewjam/saml/samlsp"
	"github.com/pkg/errors"

	"gopkg.hrry.dev/homelab/pkg/certutil"
)

type SAMLOptions struct {
	CertFile string
	KeyFile  string
	Host     string
	// Path is the base path that all saml endpoints will be served at.
	Path               string
	CookieName         string
	DefaultRedirectURI string
}

func NewSAMLService(opts *SAMLOptions) (*SAMLService, error) {
	if opts == nil {
		opts = &SAMLOptions{}
	}
	opts.defaults()
	crt, err := certutil.OpenCertificate(opts.CertFile)
	if err != nil {
		return nil, err
	}
	key, err := certutil.OpenKey(opts.KeyFile)
	if err != nil {
		return nil, err
	}
	if !certutil.IsRSA(key) {
		return nil, errors.New("expected RSA key")
	}
	mw, err := samlsp.New(samlsp.Options{
		SignRequest:       true,
		AllowIDPInitiated: true,
		URL: url.URL{
			Scheme: "https",
			Host:   opts.Host,
			Path:   opts.Path,
		},
		Key:                key.(*rsa.PrivateKey),
		Certificate:        crt,
		CookieName:         opts.CookieName,
		DefaultRedirectURI: opts.DefaultRedirectURI,
		CookieSameSite:     http.SameSiteLaxMode,
	})
	if err != nil {
		return nil, err
	}
	return &SAMLService{mw}, nil
}

type SAMLController interface {
	Logout(w http.ResponseWriter, r *http.Request)
}

type SAMLService struct{ mw *samlsp.Middleware }

func (ss *SAMLService) Logout(w http.ResponseWriter, r *http.Request) {
	panic("logout unimplemented")
}

func (ss *SAMLService) SloURL() url.URL {
	return ss.mw.ServiceProvider.SloURL
}

func (ss *SAMLService) AcsURL() url.URL {
	return ss.mw.ServiceProvider.AcsURL
}

func (ss *SAMLService) MetadataURL() url.URL {
	return ss.mw.ServiceProvider.MetadataURL
}

func (ss *SAMLService) Middleware() http.Handler {
	return ss.mw
}

func (so *SAMLOptions) defaults() {
	type KP struct {
		key   string
		val   *string
		deflt string
	}
	for _, kv := range []KP{
		{"API_SAML_PATH", &so.Path, "/api/"},
		{"API_SAML_HOST", &so.Host, ""},
		{"API_SAML_COOKIE_NAME", &so.CookieName, "samltoken"},
		{"API_SAML_CERT_FILE", &so.CertFile, ""},
		{"API_SAML_KEY_FILE", &so.KeyFile, ""},
		{"API_SAML_REDIRECT_URI", &so.DefaultRedirectURI, ""},
	} {
		if len(*kv.val) == 0 {
			*kv.val = getenv(kv.key, kv.deflt)
		}
	}
}
