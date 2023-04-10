package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"harrybrown.com/pkg/web"
)

const (
	Issuer          = "harrybrwn.com"
	TokenAudience   = "user"
	refreshAudience = "refresh"
)

var (
	ErrNoAudience     = errors.New("no token audience")
	ErrTokenExpired   = jwt.NewValidationError("token expired", jwt.ValidationErrorExpired)
	ErrBadIssuerOrAud = jwt.NewValidationError(
		"invalid issuer or audience",
		jwt.ValidationErrorAudience|jwt.ValidationErrorIssuer,
	)
	ErrBadRefreshTokenAud = errors.New("bad refresh token audience")
	ErrBadIssuer          = errors.New("bad token issuer")
	ErrNoClaims           = errors.New("no claims found")
)

type AuthContextKey string

const (
	ClaimsContextKey = AuthContextKey("jwt-ctx-claims")
	TokenContextKey  = AuthContextKey("jwt-ctx-token")
)

type getter interface {
	Get(string) interface{}
}

func GetClaims(g getter) *Claims {
	val := g.Get(string(ClaimsContextKey))
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
	jwt.RegisteredClaims
}

type TokenConfig interface {
	Private() crypto.PrivateKey
	Public() crypto.PublicKey
	Type() jwt.SigningMethod

	GetToken(*http.Request) (string, error)
}

func GuardMiddleware(conf TokenConfig) echo.MiddlewareFunc {
	keyfunc := func(*jwt.Token) (interface{}, error) {
		return conf.Public(), nil
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			auth, err := conf.GetToken(req)
			if err != nil {
				return echo.ErrUnauthorized.SetInternal(
					errors.Wrap(err, "could not get token from request"),
				)
			}
			var claims Claims
			token, err := jwt.ParseWithClaims(auth, &claims, keyfunc)
			if err != nil {
				return echo.ErrUnauthorized.SetInternal(err)
			}
			err = isValid(token, &claims)
			if err != nil {
				return err
			}
			c.Set(string(ClaimsContextKey), &claims)
			return next(c)
		}
	}
}

func Guard(conf TokenConfig) func(h http.Handler) http.Handler {
	keyFunc := func(*jwt.Token) (interface{}, error) {
		return conf.Public(), nil
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth, err := conf.GetToken(r)
			if err != nil {
				web.WriteError(w, web.StatusError(http.StatusUnauthorized, err))
				return
			}
			var claims Claims
			token, err := jwt.ParseWithClaims(auth, &claims, keyFunc)
			if err != nil {
				web.WriteError(w, web.StatusError(http.StatusUnauthorized, err, "invalid token"))
				return
			}
			err = isValid(token, &claims)
			if err != nil {
				web.WriteError(w, web.WrapError(err))
				return
			}
			ctx := context.WithValue(r.Context(), ClaimsContextKey, &claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ImplicitUser will look for an auth token and store a partial user in the
// request context if one is found.
func ImplicitUser(conf TokenConfig) echo.MiddlewareFunc {
	keyfunc := func(*jwt.Token) (interface{}, error) {
		return conf.Public(), nil
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			auth, err := conf.GetToken(req)
			if err != nil {
				logger.Info("ImplicitUser: no token found")
				return next(c)
			}
			var claims Claims
			token, err := jwt.ParseWithClaims(auth, &claims, keyfunc)
			if err != nil {
				logger.Warn("ImplicitUser: failed to token from claims")
				return next(c)
			}
			err = isValid(token, &claims)
			if err == nil {
				c.Set(string(ClaimsContextKey), &claims)
			}
			return next(c)
		}
	}
}

func isValid(token *jwt.Token, claims *Claims) error {
	if !token.Valid {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "invalid token"}
	}
	if len(claims.Audience) == 0 {
		return echo.ErrBadRequest.SetInternal(ErrNoAudience)
	}
	if claims.Issuer != Issuer || claims.Audience[0] != TokenAudience {
		return echo.ErrBadRequest.SetInternal(ErrBadIssuerOrAud)
	}
	return nil
}

func ValidateRefreshToken(token string, keyfunc func(*jwt.Token) (interface{}, error)) (*Claims, error) {
	var claims Claims
	tok, err := jwt.ParseWithClaims(token, &claims, keyfunc)
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Message: "invalid refresh token",
		}
	}
	if len(claims.Audience) < 1 || claims.Audience[0] != refreshAudience {
		return nil, ErrBadRefreshTokenAud
	}
	if claims.Issuer != Issuer {
		return nil, errors.Wrapf(ErrBadIssuer, "%s", claims.Issuer)
	}
	return &claims, nil
}

var (
	Expiration        = time.Hour * 2
	RefreshExpiration = time.Hour * 24 * 5
	JWTScheme         = "Bearer"
)

type TokenResponse struct {
	Token        string           `json:"token"`
	Expires      *jwt.NumericDate `json:"expires"`
	TokenType    string           `json:"token_type"`
	RefreshToken string           `json:"refresh_token"`
}

func NewTokenResponse(
	conf TokenConfig,
	claims *Claims,
) (*TokenResponse, error) {
	now := time.Now()
	resp, err := newAccessToken(conf, now, claims)
	if err != nil {
		return nil, err
	}
	err = resp.initRefreshToken(conf, now, *claims)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func NewAccessToken(conf TokenConfig, claims *Claims) (*TokenResponse, error) {
	now := time.Now()
	resp, err := newAccessToken(conf, now, claims)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func newAccessToken(conf TokenConfig, now time.Time, claims *Claims) (*TokenResponse, error) {
	expires := now.Add(Expiration)
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(expires)
	tok := jwt.NewWithClaims(conf.Type(), claims)
	token, err := tok.SignedString(conf.Private())
	if err != nil {
		return nil, err
	}
	return &TokenResponse{
		Token:     token,
		Expires:   claims.ExpiresAt,
		TokenType: JWTScheme,
	}, nil
}

func (tr *TokenResponse) initRefreshToken(conf TokenConfig, now time.Time, c Claims) error {
	c.Audience = []string{refreshAudience}
	c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(RefreshExpiration))
	tok := jwt.NewWithClaims(conf.Type(), &c)
	refresh, err := tok.SignedString(conf.Private())
	if err != nil {
		return err
	}
	tr.RefreshToken = refresh
	return nil
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
