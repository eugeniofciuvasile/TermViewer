package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/config"
	dbPkg "github.com/eugen/termviewer/server/backend/pkg/db"
	"github.com/eugen/termviewer/server/backend/pkg/models"
	"github.com/eugen/termviewer/server/backend/pkg/tokens"
	"github.com/gofiber/fiber/v2"
)

type ShareSessionHandler struct {
	cfg *config.Config
}

func InitShareSessionHandler(cfg *config.Config) *ShareSessionHandler {
	return &ShareSessionHandler{
		cfg: cfg,
	}
}


type ConnectShareSessionRequest struct {
	SessionToken string `json:"session_token"`
}

type ConnectShareSessionResponse struct {
	SessionID            uint      `json:"session_id"`
	ClientID             string    `json:"client_id"`
	Status               string    `json:"status"`
	RelayURL             string    `json:"relay_url"`
	ExpiresAt            time.Time `json:"expires_at"`
	ServerTLSFingerprint string    `json:"server_tls_fingerprint,omitempty"`
}

type RefreshShareSessionRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshShareSessionResponse struct {
	SessionID    uint      `json:"session_id"`
	Status       string    `json:"status"`
	RelayURL     string    `json:"relay_url"`
	SessionToken string    `json:"session_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (h *ShareSessionHandler) Connect(c *fiber.Ctx) error {
	db := getDB(c)
	req := new(ConnectShareSessionRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.SessionToken == "" {
		return c.Status(400).JSON(fiber.Map{"error": "session token is required"})
	}

	tokenHash := tokens.HashToken(req.SessionToken)
	var shareSession models.ShareSession
	if err := db.Preload("Machine").Where("session_token_hash = ?", tokenHash).First(&shareSession).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "share session not found"})
	}

	now := time.Now()
	switch shareSession.Status {
	case models.ShareSessionStatusWaiting:
		if shareSession.ExpiresAt.Add(10 * time.Second).Before(now) {
			if err := db.Model(&shareSession).Updates(map[string]interface{}{
				"status":   models.ShareSessionStatusExpired,
				"ended_at": now,
			}).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to expire share session"})
			}

			if err := db.Model(&shareSession.Machine).Update("status", models.MachineStatusOnline).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to reset machine status"})
			}

			return c.Status(410).JSON(fiber.Map{"error": "share session expired"})
		}
	case models.ShareSessionStatusStreaming:
		if shareSession.ExpiresAt.Add(10 * time.Second).Before(now) {
			if err := db.Model(&shareSession).Updates(map[string]interface{}{
				"status":   models.ShareSessionStatusEnded,
				"ended_at": now,
			}).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to end share session"})
			}

			if err := db.Model(&shareSession.Machine).Update("status", models.MachineStatusOnline).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to reset machine status"})
			}

			return c.Status(410).JSON(fiber.Map{"error": "share session expired"})
		}
	default:
		return c.Status(409).JSON(fiber.Map{"error": "share session is no longer available"})
	}

	// Audit Log (Log against the machine owner)
	auditMsg := fmt.Sprintf("External user connected to machine: %s (Session ID: %d)", shareSession.Machine.Name, shareSession.ID)
	// We manually set userID for the audit since the current request might not have one in Locals
	c.Locals("userID", shareSession.UserID)
	dbPkg.LogActivity(c, db, "SHARE_SESSION_CONNECT", "session", auditMsg)

	return c.JSON(ConnectShareSessionResponse{
		SessionID:            shareSession.ID,
		ClientID:             shareSession.Machine.ClientID,
		Status:               shareSession.Status,
		RelayURL:             websocketBaseURL(c.BaseURL()) + "/ws/relay/session/" + strconv.FormatUint(uint64(shareSession.ID), 10),
		ExpiresAt:            shareSession.ExpiresAt,
		ServerTLSFingerprint: h.cfg.ServerTLSFingerprint,
	})
}

func (h *ShareSessionHandler) Refresh(c *fiber.Ctx) error {
	db := getDB(c)
	req := new(RefreshShareSessionRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.RefreshToken == "" {
		return c.Status(400).JSON(fiber.Map{"error": "refresh token is required"})
	}

	refreshTokenHash := tokens.HashToken(req.RefreshToken)
	var shareSession models.ShareSession
	if err := db.Preload("Machine").Where("refresh_token_hash = ?", refreshTokenHash).First(&shareSession).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "share session not found"})
	}

	if shareSession.Status != models.ShareSessionStatusStreaming {
		return c.Status(409).JSON(fiber.Map{"error": "share session is no longer streaming"})
	}

	now := time.Now()
	if shareSession.Machine.LastSeenAt == nil || now.Sub(*shareSession.Machine.LastSeenAt) > machinePresenceOfflineAfter {
		if err := db.Model(&shareSession).Updates(map[string]interface{}{
			"status":   models.ShareSessionStatusEnded,
			"ended_at": now,
		}).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to end share session"})
		}

		if err := db.Model(&shareSession.Machine).Update("status", models.MachineStatusOffline).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to reset machine status"})
		}

		return c.Status(410).JSON(fiber.Map{"error": "share session expired"})
	}

	sessionToken, err := rotateStreamingShareSessionTokens(db, &shareSession, defaultShareSessionTTL, now)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to refresh share session"})
	}

	// Audit Log
	c.Locals("userID", shareSession.UserID)
	dbPkg.LogActivity(c, db, "SHARE_SESSION_REFRESH", "session", fmt.Sprintf("Refreshed session for machine: %s", shareSession.Machine.Name))

	return c.JSON(RefreshShareSessionResponse{
		SessionID:    shareSession.ID,
		Status:       shareSession.Status,
		RelayURL:     websocketBaseURL(c.BaseURL()) + "/ws/relay/session/" + strconv.FormatUint(uint64(shareSession.ID), 10),
		SessionToken: sessionToken,
		RefreshToken: req.RefreshToken,
		ExpiresAt:    shareSession.ExpiresAt,
	})
}

type HeartbeatShareSessionResponse struct {
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *ShareSessionHandler) Heartbeat(c *fiber.Ctx) error {
	db := getDB(c)
	sessionID := c.Locals("sessionID").(uint)

	var shareSession models.ShareSession
	if err := db.First(&shareSession, sessionID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "session not found"})
	}

	if shareSession.Status != models.ShareSessionStatusStreaming {
		return c.Status(409).JSON(fiber.Map{"error": "session is not active"})
	}

	now := time.Now()
	shareSession.ExpiresAt = now.Add(defaultShareSessionTTL)
	if err := db.Model(&shareSession).Update("expires_at", shareSession.ExpiresAt).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update heartbeat"})
	}

	return c.JSON(HeartbeatShareSessionResponse{
		ExpiresAt: shareSession.ExpiresAt,
	})
}
