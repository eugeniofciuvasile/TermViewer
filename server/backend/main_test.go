package main

import (
	"net/http/httptest"
	"testing"

	"github.com/eugen/termviewer/server/backend/pkg/config"
	"github.com/eugen/termviewer/server/backend/pkg/handlers"
	"github.com/eugen/termviewer/server/backend/pkg/relay"
	"github.com/eugen/termviewer/server/backend/pkg/routes"
	"github.com/gofiber/fiber/v2"
)

func TestPreflightRegisterRequest(t *testing.T) {
	app := newTestApp("http://localhost:3000")

	req := httptest.NewRequest(fiber.MethodOptions, "/api/register", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", fiber.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("preflight request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected %d, got %d", fiber.StatusNoContent, resp.StatusCode)
	}

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected Access-Control-Allow-Origin header to echo origin, got %q", got)
	}
}

func TestPreflightProtectedMachineRequestBypassesAuth(t *testing.T) {
	app := newTestApp("http://localhost:3000")

	req := httptest.NewRequest(fiber.MethodOptions, "/api/machines", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", fiber.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "authorization")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("preflight request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("expected %d, got %d", fiber.StatusNoContent, resp.StatusCode)
	}

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Fatal("expected Access-Control-Allow-Headers to be present")
	}
}

func newTestApp(allowedOrigins string) *fiber.App {
	cfg := &config.Config{
		CORSAllowedOrigins: allowedOrigins,
		KeycloakAdminRole:  "termviewer-admin",
	}

	authHandler := handlers.InitAuthHandler(nil, nil, "http://localhost:3000")
	agentHandler := handlers.InitAgentHandler()
	machineHandler := handlers.InitMachineHandler()
	shareSessionHandler := handlers.InitShareSessionHandler(cfg)
	re := relay.InitRelayEngine()

	app := fiber.New()
	routes.SetupRoutes(app, cfg, authHandler, agentHandler, machineHandler, shareSessionHandler, nil, re)
	return app
}
