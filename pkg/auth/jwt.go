package auth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"database/sql/driver"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var (
	ErrTokenExpired = jwt.NewValidationError("token expired", jwt.ValidationErrorExpired)
)

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleDefault Role = "default"

	ClaimsContextKey = "jwt-ctx-claims"
	TokenContextKey  = "jwt-ctx-token"
)

func (r *Role) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		*r = Role(v)
	case []uint8:
		*r = Role(v)
	default:
		return errors.New("unknown type cannot become type auth.Role")
	}
	return nil
}

func (r *Role) Value() (driver.Value, error) {
	return string(*r), nil
}

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
			auth, err := conf.GetToken(c.Request())
			if err != nil {
				return err
			}
			var claims Claims
			token, err := jwt.ParseWithClaims(auth, &claims, keyfunc)
			if err != nil {
				return err
			}
			if !token.Valid {
				return errors.New("invalid token")
			}
			if now.After(time.Unix(claims.ExpiresAt, 0)) {
				return ErrTokenExpired
			}
			c.Set(ClaimsContextKey, &claims)
			return next(c)
		}
	}
}

var (
	SigningMethod     = jwt.SigningMethodES256
	Expiration        = time.Hour
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
	tok := jwt.NewWithClaims(conf.Type(), claims)
	token, err := tok.SignedString(key)
	if err != nil {
		return nil, err
	}
	claims.Audience = "refresh"
	claims.ExpiresAt = now.Add(RefreshExpiration).Unix()
	tok = jwt.NewWithClaims(conf.Type(), claims)
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
	// key ed25519.PrivateKey
	// pub ed25519.PublicKey
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
