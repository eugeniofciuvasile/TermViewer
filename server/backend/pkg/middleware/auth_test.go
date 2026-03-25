package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type fakeRealmRoleChecker struct {
	hasRole bool
	err     error
}

func (f fakeRealmRoleChecker) UserHasRealmRole(userID, roleName string) (bool, error) {
	return f.hasRole, f.err
}

func TestRequireRealmRoleAllowsAdmin(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("keycloakID", "user-123")
		return c.Next()
	})
	app.Get("/", RequireRealmRole(fakeRealmRoleChecker{hasRole: true}, "termviewer-admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestRequireRealmRoleRejectsNonAdmin(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("keycloakID", "user-123")
		return c.Next()
	})
	app.Get("/", RequireRealmRole(fakeRealmRoleChecker{hasRole: false}, "termviewer-admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected %d, got %d", fiber.StatusForbidden, resp.StatusCode)
	}
}

func TestRequireRealmRoleHandlesVerifierFailure(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("keycloakID", "user-123")
		return c.Next()
	})
	app.Get("/", RequireRealmRole(fakeRealmRoleChecker{err: errors.New("boom")}, "termviewer-admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", fiber.StatusInternalServerError, resp.StatusCode)
	}
}
