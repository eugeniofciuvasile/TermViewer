package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	dbPkg "github.com/eugen/termviewer/server/backend/pkg/db"
	"github.com/eugen/termviewer/server/backend/pkg/models"
	"github.com/eugen/termviewer/server/backend/pkg/tokens"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type MachineHandler struct {
}

func InitMachineHandler() *MachineHandler {
	return &MachineHandler{}
}

type CreateMachineRequest struct {
	Name string `json:"name"`
}

type UpdateMachineRequest struct {
	Name string `json:"name"`
}

type CreateMachineResponse struct {
	Name         string `json:"name"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"` // Only shown once
}

type RegenerateSecretResponse struct {
	ClientSecret string `json:"client_secret"` // Only shown once
}

type MachineShareSessionResponse struct {
	ID        uint      `json:"id"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}

type MachineResponse struct {
	ID                 uint                         `json:"id"`
	Name               string                       `json:"name"`
	ClientID           string                       `json:"client_id"`
	Status             string                       `json:"status"`
	LastSeenAt         *time.Time                   `json:"last_seen_at"`
	ActiveShareSession *MachineShareSessionResponse `json:"active_share_session,omitempty"`
}

type CreateShareSessionRequest struct {
	TTLSeconds int `json:"ttl_seconds"`
}

type CreateShareSessionResponse struct {
	SessionID    uint      `json:"session_id"`
	SessionToken string    `json:"session_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	ServerURL    string    `json:"server_url"`
	DeepLink     string    `json:"deep_link"`
	Status       string    `json:"status"`
}

func (h *MachineHandler) CreateMachine(c *fiber.Ctx) error {
	user := c.Locals("user").(*models.User)
	db := getDB(c)

	req := new(CreateMachineRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	clientID, err := tokens.GenerateSecureToken(16)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate client id"})
	}

	clientSecret, err := tokens.GenerateSecureToken(32)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate client secret"})
	}

	// Hash the secret
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to hash secret"})
	}

	machine := models.Machine{
		UserID:       user.ID,
		Name:         req.Name,
		ClientID:     clientID,
		ClientSecret: string(hashedSecret),
		Status:       models.MachineStatusOffline,
	}

	if err := db.Create(&machine).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create machine in database"})
	}

	// Audit Log the creation
	dbPkg.LogActivity(c, db, "MACHINE_CREATE", "machine", fmt.Sprintf("Created machine: %s", machine.Name))

	return c.Status(201).JSON(CreateMachineResponse{
		Name:         machine.Name,
		ClientID:     machine.ClientID,
		ClientSecret: clientSecret,
	})
}

func (h *MachineHandler) ListMachines(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	db := getDB(c)

	var machines []models.Machine
	if err := db.Where("user_id = ?", userID).Find(&machines).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch machines"})
	}

	now := time.Now()
	response := make([]MachineResponse, 0, len(machines))

	for i := range machines {
		if err := syncMachinePresence(db, &machines[i], now); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to update machine presence"})
		}

		activeShare, err := getActiveShareSession(db, machines[i].ID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to fetch machine share state"})
		}

		item := MachineResponse{
			ID:         machines[i].ID,
			Name:       machines[i].Name,
			ClientID:   machines[i].ClientID,
			Status:     machines[i].Status,
			LastSeenAt: machines[i].LastSeenAt,
		}

		if activeShare != nil {
			item.ActiveShareSession = &MachineShareSessionResponse{
				ID:        activeShare.ID,
				Status:    activeShare.Status,
				ExpiresAt: activeShare.ExpiresAt,
			}
		}

		response = append(response, item)
	}

	return c.JSON(response)
}

func (h *MachineHandler) UpdateMachine(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	db := getDB(c)

	machineID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid machine id"})
	}

	req := new(UpdateMachineRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	var machine models.Machine
	if err := db.Where("id = ? AND user_id = ?", machineID, userID).First(&machine).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "machine not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to load machine"})
	}

	if err := db.Model(&machine).Update("name", req.Name).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update machine name"})
	}

	// Audit Log
	dbPkg.LogActivity(c, db, "MACHINE_UPDATE", "machine", fmt.Sprintf("Updated machine name to: %s (ID: %d)", machine.Name, machine.ID))

	return c.SendStatus(204)
}

func (h *MachineHandler) RegenerateMachineSecret(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	db := getDB(c)

	machineID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid machine id"})
	}

	var machine models.Machine
	if err := db.Where("id = ? AND user_id = ?", machineID, userID).First(&machine).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "machine not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to load machine"})
	}

	clientSecret, err := tokens.GenerateSecureToken(32)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate client secret"})
	}

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to hash secret"})
	}

	if err := db.Model(&machine).Update("client_secret", string(hashedSecret)).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update secret"})
	}

	// Audit Log
	dbPkg.LogActivity(c, db, "MACHINE_SECRET_REGENERATE", "machine", fmt.Sprintf("Regenerated secret for machine: %s (ID: %d)", machine.Name, machine.ID))

	return c.JSON(RegenerateSecretResponse{
		ClientSecret: clientSecret,
	})
}

func (h *MachineHandler) DeleteMachine(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	db := getDB(c)

	machineID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid machine id"})
	}

	var machine models.Machine
	if err := db.Where("id = ? AND user_id = ?", machineID, userID).First(&machine).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "machine not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to load machine"})
	}

	// Delete the machine (and ideally cascade related share sessions if DB configured, or manually)
	if err := db.Delete(&machine).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete machine"})
	}

	// Audit Log
	dbPkg.LogActivity(c, db, "MACHINE_DELETE", "machine", fmt.Sprintf("Deleted machine: %s (ID: %d)", machine.Name, machineID))

	return c.SendStatus(204)
}

func (h *MachineHandler) CreateShareSession(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)
	db := getDB(c)

	machineID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid machine id"})
	}

	var machine models.Machine
	if err := db.Where("id = ? AND user_id = ?", machineID, userID).First(&machine).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "machine not found"})
		}

		return c.Status(500).JSON(fiber.Map{"error": "failed to load machine"})
	}

	req := new(CreateShareSessionRequest)
	if len(c.Body()) > 0 {
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
	}

	now := time.Now()
	if err := syncMachinePresence(db, &machine, now); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update machine presence"})
	}

	if machine.Status == models.MachineStatusOffline {
		return c.Status(409).JSON(fiber.Map{"error": "machine is offline"})
	}

	ttl := normalizeShareSessionTTL(req.TTLSeconds)
	shareSession, tokenPair, err := createOrRotateWaitingShareSession(db, &machine, ttl, now)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create share session"})
	}

	// Audit Log
	dbPkg.LogActivity(c, db, "SHARE_SESSION_CREATE", "session", fmt.Sprintf("Created share session for machine: %s (Session ID: %d)", machine.Name, shareSession.ID))

	serverURL := c.BaseURL()
	return c.JSON(CreateShareSessionResponse{
		SessionID:    shareSession.ID,
		SessionToken: tokenPair.SessionToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    shareSession.ExpiresAt,
		ServerURL:    serverURL,
		DeepLink:     buildShareDeepLink(serverURL, tokenPair.SessionToken, tokenPair.RefreshToken),
		Status:       shareSession.Status,
	})
}
