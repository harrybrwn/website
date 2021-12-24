package auth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

const (
	Issuer        = "harrybrwn.com"
	TokenAudience = "user"
)

var (
	ErrTokenExpired = jwt.NewValidationError("token expired", jwt.ValidationErrorExpired)
)

type getter interface {
	Get(string) interface{}
}

func GetClaims(g getter) *Claims {
	val := g.Get(ClaimsContextKey)
	claims, ok := val.(*Claims)
	if !ok {
		return nil
	}
	return claims
}

type Claims struct {
	ID    int       `json:"id"`
	UUID  uuid.UUID `json:"uuid"`
	Roles []Role    `json:"roles"`
	jwt.StandardClaims
}

type TokenConfig interface {
	Private() crypto.PrivateKey
	Public() crypto.PublicKey
	Type() jwt.SigningMethod

	GetToken(*http.Request) (string, error)
}

func Guard(conf TokenConfig) echo.MiddlewareFunc {
	keyfunc := func(*jwt.Token) (interface{}, error) {
		return conf.Public(), nil
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			now := time.Now().UTC()
			req := c.Request()
			auth, err := conf.GetToken(req)
			if err != nil {
				return wrap(http.StatusUnauthorized, err, "could not get token from request")
			}
			var claims Claims
			token, err := jwt.ParseWithClaims(auth, &claims, keyfunc)
			if err != nil {
				return wrap(http.StatusUnauthorized, err, "could not parse token with claims")
			}

			if !token.Valid {
				return &echo.HTTPError{Code: http.StatusBadRequest, Message: "invalid token"}
			}
			if now.After(time.Unix(claims.ExpiresAt, 0)) {
				return &echo.HTTPError{
					Code:     http.StatusUnauthorized,
					Message:  "not authorized",
					Internal: ErrTokenExpired,
				}
			}
			if claims.Issuer != Issuer || claims.Audience != TokenAudience {
				return &echo.HTTPError{
					Code:     http.StatusBadRequest,
					Message:  "bad request",
					Internal: errors.New("jwt token issuer or audience is missmatched"),
				}
			}
			c.Set(ClaimsContextKey, &claims)
			return next(c)
		}
	}
}

var (
	SigningMethod     = jwt.SigningMethodES256
	Expiration        = time.Hour * 2
	RefreshExpiration = Expiration * 12
	JWTScheme         = "Bearer"
)

type TokenResponse struct {
	Token        string `json:"token"`
	Expires      int64  `json:"expires"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func NewTokenResponse(
	conf TokenConfig,
	claims *Claims,
) (*TokenResponse, error) {
	now := time.Now()
	key := conf.Private()
	expires := now.Add(Expiration).Unix()
	claims.IssuedAt = now.Unix()
	claims.ExpiresAt = expires
	c := *claims
	tok := jwt.NewWithClaims(conf.Type(), &c)
	token, err := tok.SignedString(key)
	if err != nil {
		return nil, err
	}

	c.Audience = "refresh"
	c.ExpiresAt = now.Add(RefreshExpiration).Unix()
	tok = jwt.NewWithClaims(conf.Type(), &c)
	refresh, err := tok.SignedString(key)
	if err != nil {
		return nil, err
	}
	return &TokenResponse{
		Token:        token,
		Expires:      expires,
		RefreshToken: refresh,
		TokenType:    JWTScheme,
	}, nil
}

var errAuthHeaderTokenMissing = errors.New("token missing from authorization header")

func GetBearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if len(h) == 0 {
		return "", errAuthHeaderTokenMissing
	}
	i := strings.Index(h, JWTScheme)
	if i < 0 {
		return "", errAuthHeaderTokenMissing
	}
	return h[i+1+len(JWTScheme):], nil
}

type edDSATokenConfig struct {
	key crypto.PrivateKey
	pub crypto.PublicKey
}

func (tc *edDSATokenConfig) GetToken(r *http.Request) (string, error) {
	return GetBearerToken(r)
}

func (tc *edDSATokenConfig) Private() crypto.PrivateKey {
	return tc.key
}

func (tc *edDSATokenConfig) Public() crypto.PublicKey {
	return tc.pub
}

func (tc *edDSATokenConfig) Type() jwt.SigningMethod {
	return jwt.SigningMethodEdDSA
}

func DecodeEdDSATokenConfig(priv, pub []byte) (TokenConfig, error) {
	var (
		cfg edDSATokenConfig
		err error
	)
	cfg.key, err = jwt.ParseEdPrivateKeyFromPEM(priv)
	if err != nil {
		return nil, err
	}
	cfg.pub, err = jwt.ParseEdPublicKeyFromPEM(pub)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func EdDSATokenConfigFromSeed(seed []byte) TokenConfig {
	key := ed25519.NewKeyFromSeed(seed)
	return &edDSATokenConfig{key: key, pub: key.Public()}
}

func GenEdDSATokenConfig() TokenConfig {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return &edDSATokenConfig{key: priv, pub: pub}
}

func GenerateECDSATokenConfig() TokenConfig {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return &tokenConfig{key: k}
}

type tokenConfig struct {
	key *ecdsa.PrivateKey
}

func (c *tokenConfig) GetToken(r *http.Request) (string, error) {
	return GetBearerToken(r)
}

func (c *tokenConfig) Private() crypto.PrivateKey {
	return c.key
}

func (c *tokenConfig) Public() crypto.PublicKey {
	return &c.key.PublicKey
}

func (c *tokenConfig) Type() jwt.SigningMethod {
	return jwt.SigningMethodES256
}

func wrap(status int, err error, msg string) *echo.HTTPError {
	return &echo.HTTPError{
		Code:     status,
		Message:  http.StatusText(status),
		Internal: errors.Wrap(err, msg),
	}
}
