package handlers

import (
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/models"
	"github.com/gofiber/fiber/v2"
)

type AgentHandler struct{}

func InitAgentHandler() *AgentHandler {
	return &AgentHandler{}
}

type HeartbeatRequest struct {
	ShareEnabled bool `json:"share_enabled"`
	TTLSeconds   int  `json:"ttl_seconds"`
	ResetStatus  bool `json:"reset_status"`
}

type HeartbeatResponse struct {
	MachineID           uint       `json:"machine_id"`
	ClientID            string     `json:"client_id"`
	Status              string     `json:"status"`
	LastSeenAt          time.Time  `json:"last_seen_at"`
	ShareSessionID      *uint      `json:"share_session_id,omitempty"`
	ShareSessionStatus  string     `json:"share_session_status,omitempty"`
	ShareSessionExpires *time.Time `json:"share_session_expires_at,omitempty"`
	SessionToken        string     `json:"session_token,omitempty"`
	RefreshToken        string     `json:"refresh_token,omitempty"`
	DeepLink            string     `json:"deep_link,omitempty"`
	ServerURL           string     `json:"server_url,omitempty"`
	Refreshed           bool       `json:"refreshed"`
}

func (h *AgentHandler) Heartbeat(c *fiber.Ctx) error {
	machineID := c.Locals("machineID").(uint)
	db := getDB(c)

	var machine models.Machine
	if err := db.First(&machine, machineID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "machine not found"})
	}

	req := new(HeartbeatRequest)
	if len(c.Body()) > 0 {
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
	}

	now := time.Now()
	machine.LastSeenAt = &now
	if machine.Status == "" || machine.Status == models.MachineStatusOffline {
		machine.Status = models.MachineStatusOnline
	}

	if err := db.Model(&machine).Updates(map[string]interface{}{
		"last_seen_at": machine.LastSeenAt,
		"status":       machine.Status,
	}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to persist heartbeat"})
	}

	if req.ResetStatus && machine.Status == models.MachineStatusStreaming {
		machine.Status = models.MachineStatusWaiting
		db.Model(&machine).Update("status", machine.Status)
	}
	response := HeartbeatResponse{
		MachineID:  machine.ID,
		ClientID:   machine.ClientID,
		Status:     machine.Status,
		LastSeenAt: now,
	}

	if !req.ShareEnabled {
		if machine.Status == models.MachineStatusWaiting {
			if err := db.Model(&models.ShareSession{}).
				Where("machine_id = ? AND status = ?", machine.ID, models.ShareSessionStatusWaiting).
				Updates(map[string]interface{}{
					"status":   models.ShareSessionStatusCancelled,
					"ended_at": now,
				}).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to cancel waiting share sessions"})
			}

			machine.Status = models.MachineStatusOnline
			if err := db.Model(&machine).Update("status", machine.Status).Error; err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "failed to update machine status"})
			}
			response.Status = machine.Status
		}

		return c.JSON(response)
	}

	ttl := normalizeShareSessionTTL(req.TTLSeconds)

	if machine.Status == models.MachineStatusStreaming {
		activeShare, refreshed, err := refreshStreamingShareSessionLease(db, &machine, ttl, now)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to load active share session"})
		}

		if activeShare != nil {
			response.ShareSessionID = &activeShare.ID
			response.ShareSessionStatus = activeShare.Status
			response.ShareSessionExpires = &activeShare.ExpiresAt
		}

		response.Refreshed = refreshed
		response.Status = machine.Status
		return c.JSON(response)
	}

	shareSession, tokenPair, refreshed, err := ensureWaitingShareSession(db, &machine, ttl, now)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to ensure waiting share session"})
	}

	response.Status = machine.Status
	if shareSession != nil {
		response.ShareSessionID = &shareSession.ID
		response.ShareSessionStatus = shareSession.Status
		response.ShareSessionExpires = &shareSession.ExpiresAt
	}

	if refreshed {
		serverURL := c.BaseURL()
		response.SessionToken = tokenPair.SessionToken
		response.RefreshToken = tokenPair.RefreshToken
		response.DeepLink = buildShareDeepLink(serverURL, tokenPair.SessionToken, tokenPair.RefreshToken)
		response.ServerURL = serverURL
		response.Refreshed = true
	}

	return c.JSON(response)
}
