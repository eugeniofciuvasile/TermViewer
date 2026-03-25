package handlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	dbPkg "github.com/eugen/termviewer/server/backend/pkg/db"

	"github.com/eugen/termviewer/server/backend/pkg/email"
	"github.com/eugen/termviewer/server/backend/pkg/keycloak"
	"github.com/eugen/termviewer/server/backend/pkg/models"
	"github.com/eugen/termviewer/server/backend/pkg/tokens"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	KC           *keycloak.KeycloakClient
	EmailService *email.EmailService
	FrontendBaseURL string
}

func InitAuthHandler(kc *keycloak.KeycloakClient, es *email.EmailService, frontendBaseURL string) *AuthHandler {
	return &AuthHandler{
		KC:              kc,
		EmailService:    es,
		FrontendBaseURL: strings.TrimRight(frontendBaseURL, "/"),
	}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PendingUserResponse struct {
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	IsApproved  bool      `json:"is_approved"`
	IsActivated bool      `json:"is_activated"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	db := getDB(c)
	if h.KC == nil {
		return c.Status(500).JSON(fiber.Map{"error": "keycloak client is not configured"})
	}

	req := new(RegisterRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	// 1. Create user in Keycloak (disabled)
	kID, err := h.KC.CreateUser(req.Username, req.Email, req.Password)
	if err != nil {
		log.Printf("Registration Error: Keycloak user creation failed: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to process registration"})
	}

	// 2. Create user in our DB
	user := models.User{
		KeycloakID: kID,
		Username:   req.Username,
		Email:      req.Email,
		IsApproved: false,
	}

	if err := db.Create(&user).Error; err != nil {
		// Rollback Keycloak user creation
		_ = h.KC.DeleteUser(kID)
		return c.Status(500).JSON(fiber.Map{"error": "failed to save user to database"})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Registration successful. Please wait for admin approval.",
	})
}

func (h *AuthHandler) ListPendingUsers(c *fiber.Ctx) error {
	db := getDB(c)
	var users []models.User
	if err := db.
		Where("is_approved = ?", false).
		Order("created_at ASC").
		Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch pending users"})
	}

	response := make([]PendingUserResponse, 0, len(users))
	for _, user := range users {
		response = append(response, PendingUserResponse{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			IsApproved:  user.IsApproved,
			IsActivated: user.IsActivated,
			CreatedAt:   user.CreatedAt,
		})
	}

	return c.JSON(response)
}

func (h *AuthHandler) ApproveUser(c *fiber.Ctx) error {
	db := getDB(c)
	if h.EmailService == nil {
		return c.Status(500).JSON(fiber.Map{"error": "email service is not configured"})
	}

	userID := c.Params("id")

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	if user.IsApproved {
		return c.Status(409).JSON(fiber.Map{"error": "user is already approved"})
	}

	// 1. Generate activation token
	token, err := tokens.GenerateSecureToken(32)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate activation token"})
	}
	expires := time.Now().Add(24 * time.Hour)

	user.IsApproved = true
	user.ActivationToken = token
	user.ActivationExpires = &expires

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update user"})
	}

	// 2. Enable in Keycloak immediately upon approval
	if err := h.KC.EnableUser(user.KeycloakID); err != nil {
		log.Printf("Error: Failed to enable user %s in Keycloak: %v", user.Username, err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to enable user in Keycloak"})
	}

	// Audit Log the admin action
	dbPkg.LogActivity(c, db, "USER_APPROVE", "user", fmt.Sprintf("Admin approved user: %s (ID: %s)", user.Username, userID))

	// 3. Send activation email
	activationLink := h.FrontendBaseURL + "/activate?token=" + token
	if err := h.EmailService.SendActivationEmail(user.Email, activationLink); err != nil {
		log.Printf("Error: Failed to send activation email: %v", err)
		return c.JSON(fiber.Map{
			"message":    "User approved and enabled, but email failed to send.",
			"email_sent": false,
			"link":       activationLink,
		})
	}

	return c.JSON(fiber.Map{
		"message":    "User approved and activation email sent.",
		"email_sent": true,
	})
}

func (h *AuthHandler) ListApprovedUsers(c *fiber.Ctx) error {
	db := getDB(c)
	var users []models.User
	if err := db.
		Where("is_approved = ? AND is_activated = ?", true, false).
		Order("updated_at DESC").
		Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch approved users"})
	}

	response := make([]PendingUserResponse, 0, len(users))
	for _, user := range users {
		response = append(response, PendingUserResponse{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			IsApproved:  user.IsApproved,
			IsActivated: user.IsActivated,
			CreatedAt:   user.CreatedAt,
		})
	}

	return c.JSON(response)
}

func (h *AuthHandler) ForceActivateUser(c *fiber.Ctx) error {
	db := getDB(c)
	userID := c.Params("id")

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	if !user.IsApproved {
		return c.Status(400).JSON(fiber.Map{"error": "user must be approved before activation"})
	}

	if user.IsActivated {
		return c.Status(409).JSON(fiber.Map{"error": "user is already activated"})
	}

	// Enable in Keycloak
	if err := h.KC.EnableUser(user.KeycloakID); err != nil {
		log.Printf("Error: Failed to enable user %s in Keycloak: %v", user.Username, err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to enable user in Keycloak"})
	}

	user.IsActivated = true
	user.ActivationToken = ""
	user.ActivationExpires = nil

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update user"})
	}

	dbPkg.LogActivity(c, db, "USER_FORCE_ACTIVATE", "user", fmt.Sprintf("Admin force-activated user: %s (ID: %s)", user.Username, userID))

	return c.JSON(fiber.Map{"message": "User force-activated successfully."})
}

func (h *AuthHandler) ResendActivationEmail(c *fiber.Ctx) error {
	db := getDB(c)
	if h.EmailService == nil {
		return c.Status(500).JSON(fiber.Map{"error": "email service is not configured"})
	}

	userID := c.Params("id")

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	if !user.IsApproved {
		return c.Status(400).JSON(fiber.Map{"error": "user must be approved first"})
	}

	if user.IsActivated {
		return c.Status(409).JSON(fiber.Map{"error": "user is already activated"})
	}

	// Generate a fresh token
	token, err := tokens.GenerateSecureToken(32)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate activation token"})
	}
	expires := time.Now().Add(24 * time.Hour)

	user.ActivationToken = token
	user.ActivationExpires = &expires

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update user"})
	}

	activationLink := h.FrontendBaseURL + "/activate?token=" + token
	if err := h.EmailService.SendActivationEmail(user.Email, activationLink); err != nil {
		log.Printf("Error: Failed to resend activation email: %v", err)
		return c.JSON(fiber.Map{
			"message":    "Email failed again.",
			"email_sent": false,
			"link":       activationLink,
		})
	}

	dbPkg.LogActivity(c, db, "USER_RESEND_ACTIVATION", "user", fmt.Sprintf("Admin resent activation email for: %s (ID: %s)", user.Username, userID))

	return c.JSON(fiber.Map{
		"message":    "Activation email resent.",
		"email_sent": true,
	})
}

func (h *AuthHandler) ActivateAccount(c *fiber.Ctx) error {
	db := getDB(c)
	if h.KC == nil {
		return c.Status(500).JSON(fiber.Map{"error": "keycloak client is not configured"})
	}

	token := c.Query("token")

	var user models.User
	if err := db.Where("activation_token = ? AND activation_expires > ?", token, time.Now()).First(&user).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid or expired activation token"})
	}

	// 1. Enable in Keycloak
	err := h.KC.EnableUser(user.KeycloakID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to enable user in Keycloak"})
	}

	// 2. Mark as activated in our DB
	user.IsActivated = true
	user.ActivationToken = ""
	user.ActivationExpires = nil

	if err := db.Save(&user).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update user"})
	}

	// Audit Log the user action
	dbPkg.LogActivity(c, db, "USER_ACTIVATE", "user", "User activated account")

	return c.JSON(fiber.Map{"message": "Account activated successfully. You can now login."})
}
