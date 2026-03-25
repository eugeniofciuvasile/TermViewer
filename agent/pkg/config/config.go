package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/term"
)

type Config struct {
	PasswordHash  string `json:"password_hash"`
	RelayURL      string `json:"relay_url,omitempty"`
	ClientID      string `json:"client_id,omitempty"`
	ClientSecret  string `json:"client_secret,omitempty"`
	TLSSkipVerify bool   `json:"tls_skip_verify,omitempty"`
	Port          int    `json:"port,omitempty"`
	Command       string `json:"command,omitempty"`
}

// HashPassword generates the SHA-256 hash of a password string.
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// GetOrCreateConfig loads the config from ~/.termviewer/config.json.
// If it doesn't exist, it prompts the user for a password, hashes it, and saves it.
func GetOrCreateConfig(providedPassword string) (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home dir: %w", err)
	}

	configDir := filepath.Join(homeDir, ".termviewer")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")

	// If config exists, read it.
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
		
		// If user provided a flag, override the saved hash temporarily in memory (useful for testing/overrides)
		if providedPassword != "" {
			cfg.PasswordHash = HashPassword(providedPassword)
		}
		return &cfg, nil
	}

	// File doesn't exist.
	var password string
	if providedPassword != "" {
		password = providedPassword
	} else {
		// Prompt interactively
		fmt.Print("Enter a new password for TermViewer mobile app authentication: ")
		bytepw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("failed to read password: %w", err)
		}
		password = string(bytepw)

		fmt.Print("Confirm password: ")
		bytepwConfirm, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("failed to read password: %w", err)
		}

		if password != string(bytepwConfirm) {
			return nil, fmt.Errorf("passwords do not match")
		}

		if len(password) < 4 {
			return nil, fmt.Errorf("password must be at least 4 characters long")
		}
	}

	hashStr := HashPassword(password)

	cfg := &Config{PasswordHash: hashStr}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	slog.Info("Configuration saved", "path", configPath)
	return cfg, nil
}

// Save persists the current config to ~/.termviewer/config.json.
func (c *Config) Save() error {
	configPath, err := configFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func configFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}
	return filepath.Join(homeDir, ".termviewer", "config.json"), nil
}
