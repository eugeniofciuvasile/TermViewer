package db

import (
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/models"
	"gorm.io/gorm"
)

// GetExpiredUnactivatedUsers returns users who were approved but didn't activate in time.
func GetExpiredUnactivatedUsers(db *gorm.DB) ([]models.User, error) {
	var users []models.User
	now := time.Now()
	
	err := db.Where("is_approved = ? AND is_activated = ? AND activation_expires < ?", true, false, now).
		Find(&users).Error
		
	return users, err
}
