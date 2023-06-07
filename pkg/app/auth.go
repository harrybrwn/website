package app

import (
	"context"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	hydra "github.com/ory/hydra-client-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/log"
)

const (
	tokenKey = "_token"
	//maxCookieAge = 2147483647
)

func NewTokenConfig() auth.TokenConfig {
	hexseed, hasSeed := os.LookupEnv("JWT_SEED")
	if hasSeed {
		logger.Info("creating token config from seed")
		seed, err := hex.DecodeString(hexseed)
		if err != nil {
			panic(errors.Wrap(err, "could not decode private key seed from hex"))
		}
		return &tokenConfig{auth.EdDSATokenConfigFromSeed(seed)}
	}
	logger.Warn("generating new key pair for token config")
	return &tokenConfig{auth.GenEdDSATokenConfig()}
}

type tokenConfig struct{ auth.TokenConfig }

func (tc *tokenConfig) GetToken(r *http.Request) (string, error) {
	c, err := r.Cookie(tokenKey)
	if err != nil {
		return auth.GetBearerToken(r)
	}
	return c.Value, nil
}

type TokenService struct {
	Tokens       auth.TokenStore
	Users        UserStore
	Config       auth.TokenConfig
	HydraAdmin   hydra.AdminApi
	CookieDomain string
}

type tokenLoginBody struct {
	Login
	LoginChallenge string `json:"login_challenge"`
	Remember       bool   `json:"remember"`
}

func (ts *TokenService) Login(c echo.Context) error {
	var (
		req    = c.Request()
		ctx    = req.Context()
		logger = log.FromContext(ctx)
	)

	logger.Info("starting login request")
	body, err := ts.getLoginBody(c, req)
	if err != nil {
		return err
	}

	// Login flow
	var (
		u      *User
		claims = auth.GetClaims(c)
	)
	if claims == nil {
		u, err = ts.Users.Login(ctx, &body.Login)
	} else {
		u, err = ts.Users.Get(ctx, claims.UUID)
	}
	if err != nil {
		return echo.ErrUnauthorized.SetInternal(errors.Wrap(err, "failed to login"))
	}
	logger = logger.WithFields(logrus.Fields{
		"username": u.Username,
		"email":    u.Email,
		"user_id":  u.UUID,
	})

	logger.Info("handling login_challenge")
	var redirectTo string
	if len(body.LoginChallenge) > 0 {
		r, hydraResp, err := ts.HydraAdmin.AcceptLoginRequest(ctx).
			LoginChallenge(body.LoginChallenge).
			AcceptLoginRequest(hydra.AcceptLoginRequest{
				Subject:  u.Email,
				Remember: hydra.PtrBool(true),
				Context: map[string]any{
					"email":    u.Email,
					"uuid":     u.UUID.String(),
					"username": u.Username,
					"roles":    u.Roles,
				},
			}).
			Execute()
		if err != nil {
			logger.WithError(err).Error("failed to accept login request")
			return &echo.HTTPError{
				Code:     hydraResp.StatusCode,
				Message:  http.StatusText(hydraResp.StatusCode),
				Internal: err,
			}
		}
		defer hydraResp.Body.Close()
		redirectTo = r.GetRedirectTo()
		logger.Infof("redirecting to %s", redirectTo)
	} else {
		logger.Warn("did not get login challenge")
	}

	if claims == nil {
		claims = u.NewClaims()
	}
	// Generate both a new access token and refresh token.
	resp, err := ts.responseToken(ctx, c, claims)
	if err != nil {
		return err
	}
	ts.setTokenCookie(c.Response(), resp, claims)
	return c.JSON(200, map[string]any{
		"redirect_to": redirectTo,
	})
}

func ConsentHandler(admin hydra.AdminApi, users UserStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			body struct {
				Challenge string `json:"consent_challenge" query:"consent_challenge"`
			}
			ctx    = c.Request().Context()
			logger = log.FromContext(ctx)
		)
		err := c.Bind(&body)
		if err != nil {
			return err
		}
		claims := auth.GetClaims(c)
		if claims == nil {
			return echo.ErrUnauthorized
		}
		logger = logger.WithFields(log.Fields{
			"id":      claims.ID,
			"uuid":    claims.UUID,
			"subject": claims.Subject,
		})
		if body.Challenge == "" {
			return echo.ErrBadRequest
		}
		u, err := users.Get(ctx, claims.UUID)
		if err != nil {
			return echo.ErrUnauthorized.SetInternal(err)
		}
		cr, hydraRes, err := admin.GetConsentRequest(ctx).ConsentChallenge(body.Challenge).Execute()
		if err != nil {
			logger.WithError(err).Error("failed to fetch consent request")
			return &echo.HTTPError{Code: hydraRes.StatusCode, Internal: err}
		}
		defer hydraRes.Body.Close()
		logger.WithFields(log.Fields{
			"skip":             *cr.Skip,
			"client.name":      cr.Client.ClientName,
			"client.client_id": cr.Client.ClientId,
		}).Info("accepting consent request")
		r, hydraConsentResp, err := admin.AcceptConsentRequest(ctx).
			ConsentChallenge(body.Challenge).
			AcceptConsentRequest(hydra.AcceptConsentRequest{
				GrantAccessTokenAudience: cr.RequestedAccessTokenAudience,
				GrantScope:               cr.RequestedScope,
				Remember:                 hydra.PtrBool(true),
				Session: &hydra.ConsentRequestSession{
					AccessToken: nil,
					IdToken: map[string]any{
						"email":   u.Email,
						"uuid":    u.UUID.String(),
						"roles":   u.Roles,
						"name":    u.Username,
						"picture": "https://hrry.me/favicon.ico", // needed by some services
					},
				},
			}).
			Execute()
		if err != nil {
			logger.Error("failed to accept consent request")
			return echo.ErrInternalServerError
		}
		defer hydraConsentResp.Body.Close()
		return c.JSON(200, map[string]any{"redirect_to": r.RedirectTo})
	}
}

func (ts *TokenService) Token(c echo.Context) error {
	var (
		req    = c.Request()
		ctx    = req.Context()
		logger = log.FromContext(ctx)
	)
	body, err := ts.getLoginBody(c, req)
	if err != nil {
		return err
	}
	logger = logger.WithFields(logrus.Fields{
		"username": body.Username,
		"email":    body.Email,
	})
	setCookie, err := ts.parserCookieQuery(req)
	if err != nil {
		return err
	}
	logger.Info("getting token")
	u, err := ts.Users.Login(ctx, &body.Login)
	if err != nil {
		return echo.ErrNotFound.SetInternal(errors.Wrap(err, "failed to login"))
	}

	if len(body.LoginChallenge) > 0 {
		_, acceptLoginResp, err := ts.HydraAdmin.AcceptLoginRequest(ctx).
			LoginChallenge(body.LoginChallenge).
			Execute()
		if err != nil {
			return echo.ErrInternalServerError
		}
		defer acceptLoginResp.Body.Close()
	}
	claims := u.NewClaims()
	// Generate both a new access token and refresh token.
	resp, err := ts.responseToken(ctx, c, claims)
	if err != nil {
		return err
	}
	if setCookie {
		ts.setTokenCookie(c.Response(), resp, claims)
	}
	return c.JSON(200, resp)
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (ts *TokenService) Refresh(c echo.Context) error {
	var (
		err      error
		req      = c.Request()
		ctx      = req.Context()
		logger   = log.FromContext(ctx)
		tokenReq RefreshTokenReq
	)
	err = c.Bind(&tokenReq)
	if err != nil {
		return echo.ErrBadRequest.SetInternal(err)
	}
	setCookie, err := ts.parserCookieQuery(req)
	if err != nil {
		return err
	}

	refreshClaims, err := auth.ValidateRefreshToken(tokenReq.RefreshToken, ts.keyfunc)
	if err != nil {
		return echo.ErrBadRequest.SetInternal(err)
	}

	stored, err := ts.Tokens.Get(ctx, refreshClaims.ID)
	if err != nil {
		return echo.ErrUnauthorized.SetInternal(err)
	}
	if tokenReq.RefreshToken != stored {
		logger.Warn("refresh token did not match up with token in storage")
		return echo.ErrUnauthorized
	}

	now := time.Now()
	claims := auth.Claims{
		ID:    refreshClaims.ID,
		UUID:  refreshClaims.UUID,
		Roles: refreshClaims.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  []string{auth.TokenAudience},
			Issuer:    auth.Issuer,
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	// Only generate a new access token, the client should still have the refresh token.
	resp, err := auth.NewAccessToken(ts.Config, &claims)
	if err != nil {
		return echo.ErrInternalServerError.SetInternal(err)
	}
	// Send back the same refresh token they sent
	resp.RefreshToken = tokenReq.RefreshToken

	c.Set(string(auth.ClaimsContextKey), claims)
	if setCookie {
		ts.setTokenCookie(c.Response(), resp, &claims)
	}
	return c.JSON(200, resp)
}

func (ts *TokenService) getLoginBody(c echo.Context, req *http.Request) (*tokenLoginBody, error) {
	var (
		err    error
		body   tokenLoginBody
		binder echo.DefaultBinder
	)
	switch err = binder.BindBody(c, &body); err {
	case nil:
		break
	case echo.ErrUnsupportedMediaType:
		logger.WithField("content-type", req.Header.Get("Content-Type")).Error("unsupported content type")
		return nil, err
	default:
		err = errors.Wrap(err, "failed to bind user data")
		return nil, echo.ErrInternalServerError.SetInternal(err)
	}
	return &body, nil
}

func (ts *TokenService) responseToken(ctx context.Context, c echo.Context, claims *auth.Claims) (*auth.TokenResponse, error) {
	resp, err := auth.NewTokenResponse(ts.Config, claims)
	if err != nil {
		return nil, echo.ErrInternalServerError.SetInternal(
			errors.Wrap(err, "could not create token response"))
	}
	err = ts.Tokens.Set(ctx, claims.ID, resp.RefreshToken)
	if err != nil {
		return nil, echo.ErrInternalServerError.SetInternal(err)
	}
	c.Set(string(auth.ClaimsContextKey), claims)
	return resp, nil
}

func (ts *TokenService) parserCookieQuery(req *http.Request) (bool, error) {
	var (
		set         bool
		err         error
		cookieQuery = req.URL.Query().Get("cookie")
	)
	if len(cookieQuery) > 0 {
		set, err = strconv.ParseBool(cookieQuery)
		if err != nil {
			return false, echo.ErrBadRequest.SetInternal(err)
		}
	} else {
		set = false
	}
	return set, nil
}

func (ts *TokenService) setTokenCookie(response http.ResponseWriter, token *auth.TokenResponse, claims *auth.Claims) {
	http.SetCookie(response, &http.Cookie{
		Name:     tokenKey,
		Value:    token.Token,
		Expires:  claims.ExpiresAt.Time,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
		Domain:   ts.CookieDomain,
		Secure:   true,
	})
}

func (ts *TokenService) Revoke(c echo.Context) error {
	var (
		ctx = c.Request().Context()
		req RefreshTokenReq
	)
	err := c.Bind(&req)
	if err != nil {
		return err
	}
	claims := auth.GetClaims(c)
	if claims == nil {
		return echo.ErrBadRequest
	}
	token, err := ts.Tokens.Get(ctx, claims.ID)
	if err != nil {
		return err
	}
	if req.RefreshToken != token {
		return &echo.HTTPError{
			Code:    http.StatusBadRequest,
			Message: "incorrect refresh token",
		}
	}
	err = ts.Tokens.Del(c.Request().Context(), claims.ID)
	if err != nil {
		return echo.ErrInternalServerError
	}
	return nil
}

func (ts *TokenService) keyfunc(*jwt.Token) (interface{}, error) {
	return ts.Config.Public(), nil
}
