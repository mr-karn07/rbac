package auth

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
)

type EnforcerMiddleware struct {
	Enforcer *casbin.Enforcer
}

// NewEnforcerMiddleware initializes the middleware with the Casbin enforcer
func NewEnforcerMiddleware(enforcer *casbin.Enforcer) *EnforcerMiddleware {
	return &EnforcerMiddleware{Enforcer: enforcer}
}

// Middleware checks the user's access permissions
func (e *EnforcerMiddleware) Middleware(c *fiber.Ctx) error {
	user, err := extractUser(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).SendString("Unauthorized")
	}

	resource := c.Query("resource")

	action := determineAction(c.Method())

	// First, check if the user is an admin for the resource
	isAdmin, err := e.Enforcer.Enforce(user, resource, "admin")
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Error enforcing policy")
	}
	if isAdmin {
		return c.Next() // Admins are allowed access to all actions
	}

	// Then, check if the user has the appropriate role for the resource and action
	allowed, err := e.Enforcer.Enforce(user, resource, action)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Error enforcing policy")
	}
	if allowed {
		return c.Next()
	}

	return c.Status(http.StatusForbidden).SendString("Forbidden")
}

// determineAction maps HTTP methods to actions
func determineAction(method string) string {
	switch method {
	case http.MethodGet:
		return "viewer"
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return "editor"
	case http.MethodDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// extractUser extracts the user credentials from the Basic Auth header
func extractUser(c *fiber.Ctx) (string, error) {
	authHeader := c.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", errors.New("no basic auth header")
	}

	encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")
	credentials, err := base64.StdEncoding.DecodeString(encodedCredentials)
	if err != nil {
		return "", err
	}

	parts := strings.SplitN(string(credentials), ":", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid credentials")
	}

	return parts[0], nil
}
