package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	MachineStatusConnected    = "connected"
	MachineStatusDisconnected = "disconnected"
	MachineStatusReconnecting = "reconnecting"
	MachineStatusError        = "error"

	AuthTypePassword = "password"
	AuthTypeKey      = "key"

	SessionStatusActive     = "active"
	SessionStatusIdle       = "idle"
	SessionStatusTerminated = "terminated"
)

// BeforeCreate hook to generate UUID for all models with ID field.
func generateUUID(id *string) {
	if *id == "" {
		*id = uuid.New().String()
	}
}

// User represents the application user (single-user auth).
type User struct {
	ID                 string    `gorm:"type:text;primaryKey" json:"id"`
	Email              string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash       string    `gorm:"not null" json:"-"`
	MustChangePassword bool      `gorm:"default:true" json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	generateUUID(&u.ID)
	return nil
}

// Machine represents a remote development machine.
type Machine struct {
	ID                   string     `gorm:"type:text;primaryKey" json:"id"`
	Name                 string     `gorm:"not null" json:"name"`
	Host                 string     `gorm:"not null" json:"host"`
	Port                 int        `gorm:"default:22" json:"port"`
	AuthType             string     `gorm:"not null" json:"auth_type"` // "key" or "password"
	EncryptedCredentials []byte     `gorm:"type:bytea" json:"-"`
	HostKeyFingerprint   string     `gorm:"type:text" json:"host_key_fingerprint,omitempty"`
	Status               string     `gorm:"default:disconnected" json:"status"`
	Capabilities         string     `gorm:"type:text" json:"capabilities"` // JSON string
	LastHeartbeat        *time.Time `json:"last_heartbeat,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	Projects             []Project  `gorm:"foreignKey:MachineID" json:"projects,omitempty"`
}

func (m *Machine) BeforeCreate(tx *gorm.DB) error {
	generateUUID(&m.ID)
	return nil
}

// Project represents a project directory on a remote machine.
type Project struct {
	ID            string    `gorm:"type:text;primaryKey" json:"id"`
	MachineID     string    `gorm:"type:text;not null;index" json:"machine_id"`
	Name          string    `gorm:"not null" json:"name"`
	DirectoryPath string    `gorm:"not null" json:"directory_path"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Machine       Machine   `gorm:"foreignKey:MachineID" json:"-"`
	Sessions      []Session `gorm:"foreignKey:ProjectID" json:"sessions,omitempty"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) error {
	generateUUID(&p.ID)
	return nil
}

// Session represents a development session within a project.
type Session struct {
	ID           string        `gorm:"type:text;primaryKey" json:"id"`
	ProjectID    string        `gorm:"type:text;not null;index" json:"project_id"`
	Name         string        `gorm:"not null" json:"name"`
	Status       string        `gorm:"default:active" json:"status"`
	LastActive   *time.Time    `json:"last_active,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Project      Project       `gorm:"foreignKey:ProjectID" json:"-"`
	TerminalTabs []TerminalTab `gorm:"foreignKey:SessionID" json:"terminal_tabs,omitempty"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) error {
	generateUUID(&s.ID)
	return nil
}

// TerminalTab represents a terminal tab (tmux window) within a session.
type TerminalTab struct {
	ID              string    `gorm:"type:text;primaryKey" json:"id"`
	SessionID       string    `gorm:"type:text;not null;index" json:"session_id"`
	TmuxWindowIndex int       `json:"tmux_window_index"`
	Name            string    `gorm:"not null" json:"name"`
	CreatedAt       time.Time `json:"created_at"`
	Session         Session   `gorm:"foreignKey:SessionID" json:"-"`
}

func (t *TerminalTab) BeforeCreate(tx *gorm.DB) error {
	generateUUID(&t.ID)
	return nil
}

// AppSession represents a server-side auth session.
type AppSession struct {
	ID           string    `gorm:"type:text;primaryKey" json:"id"`
	UserID       string    `gorm:"type:text;not null;index" json:"user_id"`
	SessionToken string    `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt    time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	User         User      `gorm:"foreignKey:UserID" json:"-"`
}

func (a *AppSession) BeforeCreate(tx *gorm.DB) error {
	generateUUID(&a.ID)
	return nil
}

// AllModels returns all model types for auto-migration.
func AllModels() []interface{} {
	return []interface{}{
		&User{},
		&Machine{},
		&Project{},
		&Session{},
		&TerminalTab{},
		&AppSession{},
	}
}
