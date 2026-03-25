package middleware

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/db"
	"github.com/eugen/termviewer/server/backend/pkg/keycloak"
	"github.com/eugen/termviewer/server/backend/pkg/models"
	"github.com/eugen/termviewer/server/backend/pkg/tokens"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type realmRoleChecker interface {
	UserHasRealmRole(userID, roleName string) (bool, error)
}

func AgentAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		clientID := c.Get("X-Client-ID")
		clientSecret := c.Get("X-Client-Secret")

		if clientID == "" || clientSecret == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing client id or secret"})
		}

		var machine models.Machine
		if err := db.DB.Where("client_id = ?", clientID).First(&machine).Error; err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid client id"})
		}

		if err := bcrypt.CompareHashAndPassword([]byte(machine.ClientSecret), []byte(clientSecret)); err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid client secret"})
		}

		c.Locals("clientID", clientID)
		c.Locals("machineID", machine.ID)
		return c.Next()
	}
}

func AppAuth(kc *keycloak.KeycloakClient) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if kc == nil {
			return c.Status(500).JSON(fiber.Map{"error": "keycloak authentication is not configured"})
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "missing or invalid authorization header"})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		ctx := context.Background()

		userInfo, err := kc.Client.GetUserInfo(ctx, token, kc.Config.KeycloakRealm)
		if err != nil {
			log.Printf("Auth Error: Keycloak rejected token on %s. Error: %v", c.Path(), err)
			return c.Status(401).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		c.Locals("keycloakID", *userInfo.Sub)
		return c.Next()
	}
}

func UserMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		keycloakID, ok := c.Locals("keycloakID").(string)
		if !ok || keycloakID == "" {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized: keycloak id missing"})
		}

		var user models.User
		if err := db.DB.Where("keycloak_id = ?", keycloakID).First(&user).Error; err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "user not found"})
		}

		if !user.IsApproved {
			return c.Status(403).JSON(fiber.Map{"error": "user not approved"})
		}

		if !user.IsActivated {
			return c.Status(403).JSON(fiber.Map{"error": "user not activated"})
		}

		c.Locals("userID", user.ID)
		c.Locals("user", &user)

		return c.Next()
	}
}

func ScopedDBMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, hasUserID := c.Locals("userID").(uint)
		isAdmin, _ := c.Locals("isAdmin").(bool)

		tx := db.DB.Begin()
		if tx.Error != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to start transaction"})
		}

		// Ensure rollback on panic or error if commit wasn't called
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
				panic(r) // re-panic after rolling back
			}
		}()

		if hasUserID {
			tx.Exec(fmt.Sprintf("SET LOCAL app.current_user_id = '%d'", userID))
		}
		if isAdmin {
			tx.Exec("SET LOCAL app.is_admin = 'true'")
		}

		c.Locals("tx", tx)

		err := c.Next()

		if err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit().Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to commit transaction"})
		}

		return nil
	}
}

func RequireRealmRole(checker realmRoleChecker, roleName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		keycloakID, ok := c.Locals("keycloakID").(string)
		if !ok || keycloakID == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing authenticated user"})
		}

		hasRole, err := checker.UserHasRealmRole(keycloakID, roleName)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to verify admin role"})
		}

		if !hasRole {
			return c.Status(403).JSON(fiber.Map{"error": "admin role required"})
		}

		c.Locals("isAdmin", true)
		return c.Next()
	}
}

func ShareSessionAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID, err := strconv.ParseUint(c.Params("sessionID"), 10, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid session id"})
		}

		sessionToken := c.Get("X-Session-Token")
		if sessionToken == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing session token"})
		}

		var shareSession models.ShareSession
		if err := db.DB.Preload("Machine").First(&shareSession, sessionID).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "share session not found"})
		}

		if tokens.HashToken(sessionToken) != shareSession.SessionTokenHash {
			return c.Status(401).JSON(fiber.Map{"error": "invalid session token"})
		}

		now := time.Now()
		switch shareSession.Status {
		case models.ShareSessionStatusWaiting:
			if shareSession.ExpiresAt.Before(now) {
				if err := db.DB.Model(&shareSession).Updates(map[string]interface{}{
					"status":   models.ShareSessionStatusExpired,
					"ended_at": now,
				}).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "failed to expire share session"})
				}

				if err := db.DB.Model(&shareSession.Machine).Update("status", models.MachineStatusOnline).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "failed to update machine status"})
				}

				return c.Status(410).JSON(fiber.Map{"error": "share session expired"})
			}

			shareSession.Status = models.ShareSessionStatusStreaming
			shareSession.ConsumedAt = &now
			shareSession.StreamStartedAt = &now
			if err := db.DB.Model(&shareSession).Updates(map[string]interface{}{
				"status":            shareSession.Status,
				"consumed_at":       shareSession.ConsumedAt,
				"stream_started_at": shareSession.StreamStartedAt,
			}).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to activate share session"})
			}

			if err := db.DB.Model(&shareSession.Machine).Update("status", models.MachineStatusStreaming).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to update machine status"})
			}
		case models.ShareSessionStatusStreaming:
			if shareSession.ExpiresAt.Before(now) {
				if err := db.DB.Model(&shareSession).Updates(map[string]interface{}{
					"status":   models.ShareSessionStatusEnded,
					"ended_at": now,
				}).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "failed to end share session"})
				}

				if err := db.DB.Model(&shareSession.Machine).Update("status", models.MachineStatusOnline).Error; err != nil {
					return c.Status(500).JSON(fiber.Map{"error": "failed to update machine status"})
				}

				return c.Status(410).JSON(fiber.Map{"error": "share session expired"})
			}
		default:
			return c.Status(409).JSON(fiber.Map{"error": fmt.Sprintf("share session is %s", shareSession.Status)})
		}

		c.Locals("clientID", shareSession.Machine.ClientID)
		c.Locals("shareSessionID", shareSession.ID)
		return c.Next()
	}
}
