package handlers

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/eugen/termviewer/server/backend/pkg/models"
	"github.com/eugen/termviewer/server/backend/pkg/tokens"
	"gorm.io/gorm"
)

const (
	defaultShareSessionTTL      = 5 * time.Minute
	minShareSessionTTL          = 1 * time.Minute
	maxShareSessionTTL          = 15 * time.Minute
	shareSessionRefreshWindow   = 1 * time.Minute
	machinePresenceOfflineAfter = 45 * time.Second
)

type shareSessionTokens struct {
	SessionToken string
	RefreshToken string
}

func normalizeShareSessionTTL(ttlSeconds int) time.Duration {
	if ttlSeconds <= 0 {
		return defaultShareSessionTTL
	}

	ttl := time.Duration(ttlSeconds) * time.Second
	if ttl < minShareSessionTTL {
		return minShareSessionTTL
	}
	if ttl > maxShareSessionTTL {
		return maxShareSessionTTL
	}

	return ttl
}

func generateShareSessionTokens() (shareSessionTokens, error) {
	sessionToken, err := tokens.GenerateSecureToken(32)
	if err != nil {
		return shareSessionTokens{}, err
	}

	refreshToken, err := tokens.GenerateSecureToken(32)
	if err != nil {
		return shareSessionTokens{}, err
	}

	return shareSessionTokens{
		SessionToken: sessionToken,
		RefreshToken: refreshToken,
	}, nil
}

func applyShareSessionTokens(shareSession *models.ShareSession, tokenPair shareSessionTokens, expiresAt time.Time) {
	shareSession.SessionTokenHash = tokens.HashToken(tokenPair.SessionToken)
	shareSession.RefreshTokenHash = tokens.HashToken(tokenPair.RefreshToken)
	shareSession.ExpiresAt = expiresAt
}

func buildShareDeepLink(serverURL, sessionToken, refreshToken string) string {
	values := url.Values{}
	values.Set("server", serverURL)
	values.Set("session_token", sessionToken)
	values.Set("refresh_token", refreshToken)

	return "termviewer://connect?" + values.Encode()
}

func websocketBaseURL(serverURL string) string {
	if strings.HasPrefix(serverURL, "https://") {
		return "wss://" + strings.TrimPrefix(serverURL, "https://")
	}

	if strings.HasPrefix(serverURL, "http://") {
		return "ws://" + strings.TrimPrefix(serverURL, "http://")
	}

	return serverURL
}

func getActiveShareSession(tx *gorm.DB, machineID uint) (*models.ShareSession, error) {
	var shareSession models.ShareSession
	err := tx.Where("machine_id = ? AND status IN ?", machineID, []string{
		models.ShareSessionStatusWaiting,
		models.ShareSessionStatusStreaming,
	}).Order("created_at DESC").First(&shareSession).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &shareSession, nil
}

func expireWaitingShareSessions(tx *gorm.DB, machineID uint, now time.Time) error {
	return tx.Model(&models.ShareSession{}).
		Where("machine_id = ? AND status = ? AND expires_at <= ?", machineID, models.ShareSessionStatusWaiting, now).
		Updates(map[string]interface{}{
			"status":   models.ShareSessionStatusExpired,
			"ended_at": now,
		}).Error
}

func cancelWaitingShareSessions(tx *gorm.DB, machineID uint, now time.Time) error {
	return tx.Model(&models.ShareSession{}).
		Where("machine_id = ? AND status = ?", machineID, models.ShareSessionStatusWaiting).
		Updates(map[string]interface{}{
			"status":   models.ShareSessionStatusCancelled,
			"ended_at": now,
		}).Error
}

func endStreamingShareSessions(tx *gorm.DB, machineID uint, now time.Time) error {
	return tx.Model(&models.ShareSession{}).
		Where("machine_id = ? AND status = ?", machineID, models.ShareSessionStatusStreaming).
		Updates(map[string]interface{}{
			"status":   models.ShareSessionStatusEnded,
			"ended_at": now,
		}).Error
}

func releaseMachineShareSessions(tx *gorm.DB, machineID uint, now time.Time) error {
	if err := cancelWaitingShareSessions(tx, machineID, now); err != nil {
		return err
	}

	if err := endStreamingShareSessions(tx, machineID, now); err != nil {
		return err
	}

	return nil
}

func syncMachinePresence(tx *gorm.DB, machine *models.Machine, now time.Time) error {
	if machine.Status == "" {
		machine.Status = models.MachineStatusOffline
	}

	if err := expireWaitingShareSessions(tx, machine.ID, now); err != nil {
		return err
	}

	if machine.LastSeenAt == nil {
		if machine.Status != models.MachineStatusOffline {
			if err := releaseMachineShareSessions(tx, machine.ID, now); err != nil {
				return err
			}

			machine.Status = models.MachineStatusOffline
			return tx.Model(machine).Update("status", machine.Status).Error
		}

		return nil
	}

	if now.Sub(*machine.LastSeenAt) > machinePresenceOfflineAfter {
		if err := releaseMachineShareSessions(tx, machine.ID, now); err != nil {
			return err
		}

		machine.Status = models.MachineStatusOffline
		return tx.Model(machine).Update("status", machine.Status).Error
	}

	activeShare, err := getActiveShareSession(tx, machine.ID)
	if err != nil {
		return err
	}

	nextStatus := models.MachineStatusOnline
	if activeShare != nil {
		switch activeShare.Status {
		case models.ShareSessionStatusWaiting:
			nextStatus = models.MachineStatusWaiting
		case models.ShareSessionStatusStreaming:
			nextStatus = models.MachineStatusStreaming
		}
	}

	if machine.Status != nextStatus {
		machine.Status = nextStatus
		return tx.Model(machine).Update("status", machine.Status).Error
	}

	return nil
}

func refreshStreamingShareSessionLease(tx *gorm.DB, machine *models.Machine, ttl time.Duration, now time.Time) (*models.ShareSession, bool, error) {
	activeShare, err := getActiveShareSession(tx, machine.ID)
	if err != nil {
		return nil, false, err
	}

	if activeShare == nil {
		if machine.Status != models.MachineStatusOnline {
			machine.Status = models.MachineStatusOnline
			if err := tx.Model(machine).Update("status", machine.Status).Error; err != nil {
				return nil, false, err
			}
		}

		return nil, false, nil
	}

	if activeShare.Status != models.ShareSessionStatusStreaming {
		nextStatus := models.MachineStatusOnline
		if activeShare.Status == models.ShareSessionStatusWaiting {
			nextStatus = models.MachineStatusWaiting
		}

		if machine.Status != nextStatus {
			machine.Status = nextStatus
			if err := tx.Model(machine).Update("status", machine.Status).Error; err != nil {
				return nil, false, err
			}
		}

		return activeShare, false, nil
	}

	if machine.Status != models.MachineStatusStreaming {
		machine.Status = models.MachineStatusStreaming
		if err := tx.Model(machine).Update("status", machine.Status).Error; err != nil {
			return nil, false, err
		}
	}

	if activeShare.ExpiresAt.Sub(now) > shareSessionRefreshWindow {
		return activeShare, false, nil
	}

	activeShare.ExpiresAt = now.Add(ttl)
	if err := tx.Model(activeShare).Update("expires_at", activeShare.ExpiresAt).Error; err != nil {
		return nil, false, err
	}

	return activeShare, true, nil
}

func createOrRotateWaitingShareSession(tx *gorm.DB, machine *models.Machine, ttl time.Duration, now time.Time) (*models.ShareSession, shareSessionTokens, error) {
	if err := expireWaitingShareSessions(tx, machine.ID, now); err != nil {
		return nil, shareSessionTokens{}, err
	}

	activeShare, err := getActiveShareSession(tx, machine.ID)
	if err != nil {
		return nil, shareSessionTokens{}, err
	}

	tokenPair, err := generateShareSessionTokens()
	if err != nil {
		return nil, shareSessionTokens{}, err
	}

	expiresAt := now.Add(ttl)
	if activeShare != nil {
		// Rotate tokens but maintain ID to keep history/audit trail
		applyShareSessionTokens(activeShare, tokenPair, expiresAt)
		activeShare.ConsumedAt = nil
		activeShare.StreamStartedAt = nil
		activeShare.EndedAt = nil

		// If it was already STREAMING, we keep it as is, or we could reset to WAITING.
		// Moving it back to WAITING allows the machine status to correctly reflect 
		// that it is ready to be joined (or re-joined).
		if activeShare.Status == models.ShareSessionStatusStreaming {
			activeShare.Status = models.ShareSessionStatusWaiting
		}

		if err := tx.Save(activeShare).Error; err != nil {
			return nil, shareSessionTokens{}, err
		}

		machine.Status = models.MachineStatusWaiting
		if err := tx.Model(machine).Update("status", machine.Status).Error; err != nil {
			return nil, shareSessionTokens{}, err
		}

		return activeShare, tokenPair, nil
	}

	shareSession := &models.ShareSession{
		MachineID: machine.ID,
		UserID:    machine.UserID,
		Status:    models.ShareSessionStatusWaiting,
	}
	applyShareSessionTokens(shareSession, tokenPair, expiresAt)

	if err := tx.Create(shareSession).Error; err != nil {
		return nil, shareSessionTokens{}, err
	}

	machine.Status = models.MachineStatusWaiting
	if err := tx.Model(machine).Update("status", machine.Status).Error; err != nil {
		return nil, shareSessionTokens{}, err
	}

	return shareSession, tokenPair, nil
}

func ensureWaitingShareSession(tx *gorm.DB, machine *models.Machine, ttl time.Duration, now time.Time) (*models.ShareSession, shareSessionTokens, bool, error) {
	if err := expireWaitingShareSessions(tx, machine.ID, now); err != nil {
		return nil, shareSessionTokens{}, false, err
	}

	activeShare, err := getActiveShareSession(tx, machine.ID)
	if err != nil {
		return nil, shareSessionTokens{}, false, err
	}

	if activeShare != nil {
		if activeShare.Status == models.ShareSessionStatusStreaming {
			return activeShare, shareSessionTokens{}, false, nil
		}

		if activeShare.ExpiresAt.Sub(now) > shareSessionRefreshWindow {
			machine.Status = models.MachineStatusWaiting
			if err := tx.Model(machine).Update("status", machine.Status).Error; err != nil {
				return nil, shareSessionTokens{}, false, err
			}

			return activeShare, shareSessionTokens{}, false, nil
		}
	}

	shareSession, tokenPair, err := createOrRotateWaitingShareSession(tx, machine, ttl, now)
	if err != nil {
		return nil, shareSessionTokens{}, false, err
	}

	return shareSession, tokenPair, true, nil
}

func rotateStreamingShareSessionTokens(tx *gorm.DB, shareSession *models.ShareSession, ttl time.Duration, now time.Time) (string, error) {
	sessionToken, err := tokens.GenerateSecureToken(32)
	if err != nil {
		return "", err
	}

	shareSession.SessionTokenHash = tokens.HashToken(sessionToken)
	shareSession.ExpiresAt = now.Add(ttl)
	if err := tx.Model(shareSession).Updates(map[string]interface{}{
		"session_token_hash": shareSession.SessionTokenHash,
		"expires_at":         shareSession.ExpiresAt,
	}).Error; err != nil {
		return "", err
	}

	return sessionToken, nil
}
