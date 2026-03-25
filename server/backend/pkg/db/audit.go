package db

import (
	"log"
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// LogActivity records a user action in the audit_logs table.
// It uses the transaction from the context if available.
func LogActivity(c *fiber.Ctx, db *gorm.DB, action, resource, details string) {
	userID, _ := c.Locals("userID").(uint)
	if userID == 0 {
		return
	}

	audit := models.AuditLog{
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
		CreatedAt: time.Now(),
	}

	// We use a background goroutine so audit logging doesn't slow down the main request.
	// Since RLS is enabled, we must ensure we use a DB connection that has the userID set,
	// OR we use the global DB (db.DB) which bypasses RLS for internal system writes.
	go func() {
		if err := DB.Create(&audit).Error; err != nil {
			log.Printf("Internal Error: Failed to write audit log: %v", err)
		}
	}()
}

// StartAuditCleanupWorker runs a background task every hour to delete old logs.
func StartAuditCleanupWorker(retentionDays int) {
	if retentionDays <= 0 {
		return
	}

	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			cutoff := time.Now().AddDate(0, 0, -retentionDays)
			result := DB.Unscoped().Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
			if result.Error != nil {
				log.Printf("Cleanup Error: Failed to delete old audit logs: %v", result.Error)
			} else if result.RowsAffected > 0 {
				log.Printf("Cleanup: Removed %d audit logs older than %d days", result.RowsAffected, retentionDays)
			}
		}
	}()
}
