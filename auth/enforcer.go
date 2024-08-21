package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
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
	// Extract user information
	user, err := extractUser(c)
	if err != nil {
		return c.Status(http.StatusUnauthorized).SendString("Unauthorized")
	}

	// Extract role from fiber context header "X-User-Role"
	role := extractRole(c)

	// Extract resource starting from the first "/" up to the end of the first segment
	resource := extractResource(c)

	// Extract method from fiber context
	action := c.Method()

	// Then, check if the user has the appropriate role for the resource and action
	allowed, err := e.Enforcer.Enforce(user, role, resource, action)
	if err != nil {
		fmt.Println(err)
		return c.Status(http.StatusInternalServerError).SendString("Error enforcing policy")
	}
	if allowed {
		return c.Next()
	}

	return c.Status(http.StatusForbidden).SendString("Forbidden")
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

// extractRole role from fiber context header "X-User-Role"
func extractRole(c *fiber.Ctx) string {
	xAuthRole := c.Get("X-User-Role")
	return xAuthRole
}

// extractResource resource starting from the first "/" up to the end of the first segment
func extractResource(c *fiber.Ctx) string {
	pathSegments := strings.SplitN(c.Path(), "/", 3)
	resource := "/" + pathSegments[1]
	return resource
}
