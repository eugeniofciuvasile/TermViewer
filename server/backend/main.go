package main

import (
	"log"
	"time"

	"os"
	"os/signal"
	"syscall"

	"github.com/eugen/termviewer/server/backend/pkg/config"
	"github.com/eugen/termviewer/server/backend/pkg/db"
	"github.com/eugen/termviewer/server/backend/pkg/email"
	"github.com/eugen/termviewer/server/backend/pkg/handlers"
	"github.com/eugen/termviewer/server/backend/pkg/keycloak"
	"github.com/eugen/termviewer/server/backend/pkg/relay"
	"github.com/eugen/termviewer/server/backend/pkg/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file loaded: %v", err)
	}

	cfg := config.LoadConfig()

	if cfg.DBPassword == "" || cfg.KeycloakAdminPass == "" {
		log.Fatal("CRITICAL ERROR: Sensitive configuration (DB/Keycloak passwords) are missing. System shutdown for security.")
	}
	if cfg.SMTPHost != "" && cfg.SMTPPass == "" {
		log.Printf("WARNING: SMTP_HOST is set but SMTP_PASS is empty — disabling email delivery.")
		cfg.SMTPHost = ""
	}

	db.InitDB(cfg)
	db.StartAuditCleanupWorker(cfg.LogRetentionDays)
	kc := keycloak.InitKeycloak(cfg)
	es := email.InitEmailService(cfg)
	re := relay.InitRelayEngine()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			users, err := db.GetExpiredUnactivatedUsers(db.DB)
			if err != nil {
				log.Printf("Cleanup Error: Failed to fetch expired users: %v", err)
				continue
			}

			for _, user := range users {
				log.Printf("Cleanup: Deleting user %s due to expired activation link", user.Username)

				if err := kc.DeleteUser(user.KeycloakID); err != nil {
					log.Printf("Cleanup Error: Failed to delete user %s from Keycloak: %v", user.Username, err)
				}

				if err := db.DB.Unscoped().Delete(&user).Error; err != nil {
					log.Printf("Cleanup Error: Failed to delete user %s from database: %v", user.Username, err)
				}
			}
		}
	}()

	authHandler := handlers.InitAuthHandler(kc, es, cfg.FrontendBaseURL)
	agentHandler := handlers.InitAgentHandler()
	machineHandler := handlers.InitMachineHandler()
	shareSessionHandler := handlers.InitShareSessionHandler(cfg)

	app := fiber.New(fiber.Config{
		ReadBufferSize: 32 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			log.Printf("Unhandled Error [%s]: %v", c.Path(), err)

			return c.Status(code).JSON(fiber.Map{
				"error": "an unexpected error occurred",
			})
		},
	})
	routes.SetupRoutes(app, cfg, authHandler, agentHandler, machineHandler, shareSessionHandler, kc, re)

	log.Println("Starting TermViewer Backend on", cfg.AppAddr)

	// Start server in background
	go func() {
		if err := app.Listen(cfg.AppAddr); err != nil {
			log.Printf("Server Error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown Signal Received. Cleaning up...")

	// Shutdown Fiber (Wait for active requests)
	if err := app.Shutdown(); err != nil {
		log.Printf("Fiber Shutdown Error: %v", err)
	}

	log.Println("TermViewer Backend Stopped.")
}
