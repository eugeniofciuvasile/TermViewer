package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppAddr              string
	DatabaseURL          string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBName               string
	DBSSLMode            string
	KeycloakURL          string
	KeycloakRealm        string
	KeycloakAdminUser    string
	KeycloakAdminPass    string
	KeycloakClientID     string
	KeycloakClientSecret string
	KeycloakAppClientID  string
	KeycloakAdminRole    string
	KeycloakSkipTLSVerify bool
	LogRetentionDays     int
	SMTPHost             string
	SMTPPort             string
	SMTPUser             string
	SMTPPass             string
	SMTPFrom             string
	ServerTLSFingerprint string
	FrontendBaseURL      string
	CORSAllowedOrigins   string
	}

	func LoadConfig() *Config {
	return &Config{
		AppAddr:              getEnv("APP_ADDR", ":3001"),
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBUser:               getEnv("DB_USER", "keycloak"),
		DBPassword:           getEnv("DB_PASSWORD", "password"),
		DBName:               getEnv("DB_NAME", "termviewer"),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		KeycloakURL:          getEnv("KC_URL", "https://sso.termviewer.local"),
		KeycloakRealm:        getEnv("KC_REALM", "termviewer"),
		KeycloakAdminUser:    getEnv("KC_ADMIN_USER", "admin"),
		KeycloakAdminPass:    getEnv("KC_ADMIN_PASS", "admin"),
		KeycloakClientID:     getEnv("KC_CLIENT_ID", "admin-cli"),
		KeycloakClientSecret: getEnv("KC_CLIENT_SECRET", ""),
		KeycloakAppClientID:  getEnv("KC_APP_CLIENT_ID", "termviewer-app"),
		KeycloakAdminRole:    getEnv("KC_ADMIN_ROLE", "termviewer-admin"),
		KeycloakSkipTLSVerify: getEnv("KC_SKIP_TLS_VERIFY", "true") == "true",
		LogRetentionDays:     getEnvInt("LOG_RETENTION_DAYS", 90),
		SMTPHost:             getEnv("SMTP_HOST", ""),
		SMTPPort:             getEnv("SMTP_PORT", "587"),
		SMTPUser:             getEnv("SMTP_USER", ""),
		SMTPPass:             getEnv("SMTP_PASS", ""),
		SMTPFrom:             getEnv("SMTP_FROM", ""),
		ServerTLSFingerprint: getEnv("SERVER_TLS_FINGERPRINT", ""),
		FrontendBaseURL:      getEnv("FRONTEND_BASE_URL", "https://termviewer.local"),
		CORSAllowedOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}
	}


func (c *Config) DatabaseDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}

	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.DBHost,
		c.DBUser,
		c.DBPassword,
		c.DBName,
		c.DBPort,
		c.DBSSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var i int
	fmt.Sscanf(value, "%d", &i)
	return i
}
