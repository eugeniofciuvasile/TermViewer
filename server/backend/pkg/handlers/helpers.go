package handlers

import (
	"github.com/eugen/termviewer/server/backend/pkg/db"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func getDB(c *fiber.Ctx) *gorm.DB {
	var gormDB *gorm.DB
	if tx, ok := c.Locals("tx").(*gorm.DB); ok {
		gormDB = tx
	} else {
		gormDB = db.DB
	}
	return gormDB.WithContext(c.UserContext())
}
