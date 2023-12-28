package app

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/matryer/is"
)

func TestSAMLService(t *testing.T) {
	is := is.New(t)
	// idpMetadataURL, err := url.Parse("https://samltest.id/saml/idp")
	// is.NoErr(err)
	// idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient,
	// 	*idpMetadataURL)
	// is.NoErr(err)
	// fmt.Printf("%+v\n", idpMetadata)
	const certFile = "../../config/pki/saml/saml.crt"
	const keyFile = "../../config/pki/saml/saml.key"
	os.Setenv("API_SAML_CERT_FILE", certFile)
	os.Setenv("API_SAML_KEY_FILE", keyFile)
	os.Setenv("API_SAML_HOST", "api.hrry.test")
	svc, err := NewSAMLService(nil)
	is.NoErr(err)
	is.Equal(svc.SloURL().Path, "/api/saml/slo")
	is.Equal(svc.MetadataURL().Path, "/api/saml/metadata")
	is.Equal(svc.AcsURL().Path, "/api/saml/acs")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	svc.mw.ServeMetadata(rec, req)
	is.Equal(rec.Code, 200)
	is.Equal(rec.Header().Get("Content-Type"), "application/samlmetadata+xml")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	svc.mw.ServeACS(rec, req)
	// fmt.Println(rec.Code, http.StatusText(rec.Code))
	// fmt.Println(rec.Header())
	// fmt.Println(rec.Body.String())
}
