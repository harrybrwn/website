package auth

import (
	"database/sql/driver"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type Role uint32

const (
	RoleInvalid Role = iota
	RoleAdmin
	RoleDefault
	RoleFamily
	RoleTanya
)

var RoleNames = [...]string{
	RoleInvalid: "",
	RoleAdmin:   "admin",
	RoleDefault: "default",
	RoleFamily:  "family",
	RoleTanya:   "tanya",
}

func (r Role) String() string {
	if int(r) >= len(RoleNames) {
		return ""
	}
	return RoleNames[r]
}

func ParseRole(s string) Role {
	switch s {
	case "admin":
		return RoleAdmin
	case "default":
		return RoleDefault
	case "family":
		return RoleFamily
	case "tanya":
		return RoleTanya
	}
	return RoleInvalid
}

var (
	ErrAdminRequired = errors.New("admin access required")
	ErrInvalidRole   = errors.New("invalid role")
)

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

func IsAdmin(cl *Claims) bool {
	for _, r := range cl.Roles {
		if r == RoleAdmin {
			return true
		}
	}
	return false
}

func RoleRequired(required Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return echo.ErrForbidden
			}
			for _, r := range claims.Roles {
				if r == required {
					return next(c)
				}
			}
			return echo.ErrForbidden
		}
	}
}

func (r *Role) Scan(src interface{}) (err error) {
	var role Role
	switch v := src.(type) {
	case string:
		role = ParseRole(v)
	case []uint8:
		role = ParseRole(string(v))
	case int8:
		role = Role(v)
	case int16:
		role = Role(v)
	case int32:
		role = Role(v)
	case int64:
		role = Role(v)
	case int:
		role = Role(v)
	case uint8:
		role = Role(v)
	case uint16:
		role = Role(v)
	case uint32:
		role = Role(v)
	case uint64:
		role = Role(v)
	case uint:
		role = Role(v)
	default:
		return errors.Wrap(ErrInvalidRole, "unknown type cannot become type auth.Role")
	}
	*r = role
	if role == RoleInvalid {
		return ErrInvalidRole
	}
	return
}

func (r *Role) Value() (driver.Value, error) {
	role := *r
	if role == RoleInvalid {
		return role, ErrInvalidRole
	}
	return role, nil
}
