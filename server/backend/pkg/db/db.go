package db

import (
	"fmt"
	"log"
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/config"
	"github.com/eugen/termviewer/server/backend/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(cfg *config.Config) {
	dsn := cfg.DatabaseDSN()
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := DB.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	fmt.Println("Database connection established")

	if err := migrateShareSessionRefreshTokens(DB); err != nil {
		log.Fatalf("Failed to prepare share session refresh token migration: %v", err)
	}

	// Migrate the schema
	err = DB.AutoMigrate(&models.User{}, &models.Machine{}, &models.ShareSession{}, &models.AuditLog{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	fmt.Println("Database migration completed")

	if err := applyRLSPolicies(DB); err != nil {
		log.Fatalf("Failed to apply RLS policies: %v", err)
	}
	fmt.Println("RLS policies applied")
}

func applyRLSPolicies(db *gorm.DB) error {
	// Enable RLS on tables
	tables := []string{"users", "machines", "share_sessions", "audit_logs"}
	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf(`ALTER TABLE "%s" ENABLE ROW LEVEL SECURITY`, table)).Error; err != nil {
			return err
		}
	}

	// 1. Users Table Policies
	db.Exec(`DROP POLICY IF EXISTS user_select_policy ON "users"`)
	db.Exec(`CREATE POLICY user_select_policy ON "users" FOR SELECT USING (id::text = current_setting('app.current_user_id', true) OR current_setting('app.is_admin', true) = 'true')`)
	
	db.Exec(`DROP POLICY IF EXISTS user_insert_policy ON "users"`)
	db.Exec(`CREATE POLICY user_insert_policy ON "users" FOR INSERT WITH CHECK (true)`) // Allow registration
	
	db.Exec(`DROP POLICY IF EXISTS user_update_policy ON "users"`)
	db.Exec(`CREATE POLICY user_update_policy ON "users" FOR UPDATE USING (id::text = current_setting('app.current_user_id', true) OR current_setting('app.is_admin', true) = 'true')`)

	// 2. Machines Table Policies
	db.Exec(`DROP POLICY IF EXISTS machine_all_policy ON "machines"`)
	db.Exec(`CREATE POLICY machine_all_policy ON "machines" FOR ALL USING (user_id::text = current_setting('app.current_user_id', true) OR current_setting('app.is_admin', true) = 'true')`)

	// 3. Share Sessions Table Policies
	db.Exec(`DROP POLICY IF EXISTS share_session_all_policy ON "share_sessions"`)
	db.Exec(`CREATE POLICY share_session_all_policy ON "share_sessions" FOR ALL USING (user_id::text = current_setting('app.current_user_id', true) OR current_setting('app.is_admin', true) = 'true')`)

	// 4. Audit Logs Table Policies
	db.Exec(`DROP POLICY IF EXISTS audit_log_isolation_policy ON "audit_logs"`)
	db.Exec(`CREATE POLICY audit_log_isolation_policy ON "audit_logs" FOR ALL USING (user_id::text = current_setting('app.current_user_id', true) OR current_setting('app.is_admin', true) = 'true')`)

	return nil
}

func migrateShareSessionRefreshTokens(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.ShareSession{}) {
		return nil
	}

	if !db.Migrator().HasColumn(&models.ShareSession{}, "refresh_token_hash") {
		if err := db.Exec(`ALTER TABLE "share_sessions" ADD COLUMN "refresh_token_hash" text`).Error; err != nil {
			return err
		}
	}

	return db.Exec(`
		UPDATE "share_sessions"
		SET "refresh_token_hash" = "session_token_hash"
		WHERE "refresh_token_hash" IS NULL
	`).Error
}
