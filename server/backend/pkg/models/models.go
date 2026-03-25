package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	MachineStatusOffline   = "offline"
	MachineStatusOnline    = "online"
	MachineStatusWaiting   = "waiting"
	MachineStatusStreaming = "streaming"
)

const (
	ShareSessionStatusWaiting   = "waiting"
	ShareSessionStatusStreaming = "streaming"
	ShareSessionStatusExpired   = "expired"
	ShareSessionStatusCancelled = "cancelled"
	ShareSessionStatusEnded     = "ended"
)

type User struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	KeycloakID        string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"keycloak_id"`
	Username          string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Email             string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	IsApproved        bool           `gorm:"index;default:false" json:"is_approved"`
	IsActivated       bool           `gorm:"index;default:false" json:"is_activated"`
	ActivationToken   string         `gorm:"type:varchar(128);index" json:"-"`
	ActivationExpires *time.Time     `json:"-"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	Machines          []Machine      `gorm:"foreignKey:UserID" json:"machines"`
}

type Machine struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	UserID        uint           `gorm:"index;not null" json:"user_id"`
	Name          string         `gorm:"type:varchar(128);not null" json:"name"`
	ClientID      string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"client_id"`
	ClientSecret  string         `gorm:"type:varchar(255);not null" json:"-"` // Hashed Client Secret
	Status        string         `gorm:"type:varchar(32);index;default:'offline'" json:"status"`
	LastSeenAt    *time.Time     `json:"last_seen_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	ShareSessions []ShareSession `gorm:"foreignKey:MachineID" json:"-"`
}

type ShareSession struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	MachineID        uint           `gorm:"index;not null" json:"machine_id"`
	UserID           uint           `gorm:"index;not null" json:"user_id"`
	Status           string         `gorm:"type:varchar(32);index;default:'waiting'" json:"status"`
	SessionTokenHash string         `gorm:"index;not null" json:"-"`
	RefreshTokenHash string         `gorm:"index;not null" json:"-"`
	ExpiresAt        time.Time      `gorm:"index;not null" json:"expires_at"`
	ConsumedAt       *time.Time     `json:"consumed_at"`
	StreamStartedAt  *time.Time     `json:"stream_started_at"`
	EndedAt          *time.Time     `json:"ended_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	Machine          Machine        `gorm:"foreignKey:MachineID" json:"-"`
}

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Action    string    `gorm:"type:varchar(64);index;not null" json:"action"`
	Resource  string    `gorm:"type:varchar(64)" json:"resource"`   // e.g., "machine", "session"
	Details   string    `gorm:"type:text" json:"details"`           // JSON or description
	IPAddress string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent string    `gorm:"type:text" json:"user_agent"`
	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
}

