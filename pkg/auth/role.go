package auth

import (
	"database/sql/driver"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
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

func AdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return echo.ErrForbidden
			}
			for _, r := range claims.Roles {
				if r == RoleAdmin {
					return next(c)
				}
			}
			return echo.ErrForbidden
		}
	}
}
