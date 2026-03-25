package routes

import (
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/config"
	"github.com/eugen/termviewer/server/backend/pkg/handlers"
	"github.com/eugen/termviewer/server/backend/pkg/keycloak"
	"github.com/eugen/termviewer/server/backend/pkg/middleware"
	"github.com/eugen/termviewer/server/backend/pkg/relay"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func SetupRoutes(
	app *fiber.App,
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	agentHandler *handlers.AgentHandler,
	machineHandler *handlers.MachineHandler,
	shareSessionHandler *handlers.ShareSessionHandler,
	kc *keycloak.KeycloakClient,
	re *relay.RelayEngine,
) {
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${id} ${status} - ${method} ${path}\n",
	}))
	app.Use(recover.New())
	
	// Defense: Rate Limiter (50 requests per 10 seconds per IP)
	app.Use(limiter.New(limiter.Config{
		Max:        50,
		Expiration: 10 * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "too many requests, please slow down",
			})
		},
	}))

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowedOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Client-ID,X-Client-Secret,X-Session-Token",
		MaxAge:       3600,
	}))

	// Basic health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Public Auth Endpoints
	api := app.Group("/api")
	api.Post("/register", authHandler.Register)
	api.Get("/activate", authHandler.ActivateAccount)
	api.Post("/share-sessions/connect", shareSessionHandler.Connect)
	api.Post("/share-sessions/refresh", shareSessionHandler.Refresh)
	api.Post("/share-sessions/heartbeat", middleware.ShareSessionAuth(), shareSessionHandler.Heartbeat)

	// Admin Endpoints
	admin := api.Group("/admin", middleware.AppAuth(kc), middleware.RequireRealmRole(kc, cfg.KeycloakAdminRole), middleware.ScopedDBMiddleware())
	admin.Get("/pending-users", authHandler.ListPendingUsers)
	admin.Get("/approved-users", authHandler.ListApprovedUsers)
	admin.Post("/approve/:id", authHandler.ApproveUser)
	admin.Post("/force-activate/:id", authHandler.ForceActivateUser)
	admin.Post("/resend-activation/:id", authHandler.ResendActivationEmail)

	agent := api.Group("/agent", middleware.AgentAuth())
	agent.Post("/heartbeat", agentHandler.Heartbeat)

	// Protected Machine Endpoints
	machines := api.Group("/machines", middleware.AppAuth(kc), middleware.UserMiddleware(), middleware.ScopedDBMiddleware())
	machines.Post("/", machineHandler.CreateMachine)
	machines.Get("/", machineHandler.ListMachines)
	machines.Patch("/:id", machineHandler.UpdateMachine)
	machines.Post("/:id/regenerate-secret", machineHandler.RegenerateMachineSecret)
	machines.Delete("/:id", machineHandler.DeleteMachine)
	machines.Post("/:id/share-session", machineHandler.CreateShareSession)

	// WebSocket Relay Endpoints
	app.Get("/ws/relay/agent", middleware.AgentAuth(), websocket.New(func(c *websocket.Conn) {
		clientID := c.Locals("clientID").(string)
		re.RegisterAgent(clientID, c)
	}))

	app.Get("/ws/relay/app/:clientID", middleware.AppAuth(kc), middleware.UserMiddleware(), websocket.New(func(c *websocket.Conn) {
		clientID := c.Params("clientID")
		re.ConnectApp(clientID, c)
	}))

	app.Get("/ws/relay/session/:sessionID", middleware.ShareSessionAuth(), websocket.New(func(c *websocket.Conn) {
		clientID := c.Locals("clientID").(string)
		re.ConnectApp(clientID, c)
	}))
}
