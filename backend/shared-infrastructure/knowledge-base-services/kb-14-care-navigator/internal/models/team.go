// Package models contains domain models for KB-14 Care Navigator
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Team represents a care team in KB-14
type Team struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TeamID    string      `gorm:"uniqueIndex;size:50;not null" json:"team_id"`
	Name      string      `gorm:"size:100;not null" json:"name"`
	Type      string      `gorm:"size:50;not null;index" json:"type"` // clinical, care_coordination, outreach, administrative
	ManagerID *uuid.UUID  `gorm:"type:uuid" json:"manager_id,omitempty"`

	// Panel Attribution - PCPs whose patients this team manages
	PanelPCPs StringSlice `gorm:"column:panel_pcps;type:jsonb;default:'[]'" json:"panel_pcps,omitempty"`

	// Settings
	MaxTasksPerMember int  `gorm:"default:20" json:"max_tasks_per_member"`
	AutoAssign        bool `gorm:"default:true" json:"auto_assign"`

	Active    bool      `gorm:"default:true;index" json:"active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Members []TeamMember `gorm:"foreignKey:TeamID" json:"members,omitempty"`
}

// TableName returns the table name for Team
func (Team) TableName() string {
	return "teams"
}

// TeamMember represents a member of a care team
type TeamMember struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MemberID string    `gorm:"uniqueIndex;size:50;not null" json:"member_id"`
	UserID   string    `gorm:"size:50;not null;index" json:"user_id"`
	TeamID   uuid.UUID `gorm:"type:uuid;not null;index" json:"team_id"`
	Name     string    `gorm:"size:100;not null" json:"name"`
	Role     string    `gorm:"size:50;not null;index" json:"role"` // Physician, Nurse, Care Coordinator, etc.
	Email    string    `gorm:"size:100" json:"email,omitempty"`
	Phone    string    `gorm:"size:20" json:"phone,omitempty"`

	// Workload Management
	MaxTasks     int        `gorm:"default:20" json:"max_tasks"`
	CurrentTasks int        `gorm:"default:0" json:"current_tasks"`
	AvailableFrom *time.Time `json:"available_from,omitempty"`
	AvailableTo   *time.Time `json:"available_to,omitempty"`

	// Skills & Preferences
	Skills    StringSlice `gorm:"type:jsonb;default:'[]'" json:"skills,omitempty"`
	Languages StringSlice `gorm:"type:jsonb;default:'[]'" json:"languages,omitempty"`

	// Supervisor for escalation
	SupervisorID *uuid.UUID `gorm:"type:uuid" json:"supervisor_id,omitempty"`

	Active    bool      `gorm:"default:true;index" json:"active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationship back to Team
	Team *Team `gorm:"foreignKey:TeamID" json:"team,omitempty"`
}

// TableName returns the table name for TeamMember
func (TeamMember) TableName() string {
	return "team_members"
}

// StringSlice is a custom type for JSONB array of strings
type StringSlice []string

// Value implements the driver.Valuer interface
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("StringSlice.Scan: unsupported type")
	}
	return json.Unmarshal(bytes, s)
}

// IsAvailable checks if a team member is currently available
func (m *TeamMember) IsAvailable() bool {
	if !m.Active {
		return false
	}

	now := time.Now()

	// Check availability window
	if m.AvailableFrom != nil && now.Before(*m.AvailableFrom) {
		return false
	}
	if m.AvailableTo != nil && now.After(*m.AvailableTo) {
		return false
	}

	return true
}

// HasCapacity checks if a team member has capacity for more tasks
func (m *TeamMember) HasCapacity() bool {
	return m.CurrentTasks < m.MaxTasks
}

// GetAvailableCapacity returns the number of additional tasks the member can take
func (m *TeamMember) GetAvailableCapacity() int {
	capacity := m.MaxTasks - m.CurrentTasks
	if capacity < 0 {
		return 0
	}
	return capacity
}

// HasSkill checks if the team member has a specific skill
func (m *TeamMember) HasSkill(skill string) bool {
	for _, s := range m.Skills {
		if s == skill {
			return true
		}
	}
	return false
}

// CreateTeamRequest represents the request body for creating a team
type CreateTeamRequest struct {
	TeamID            string   `json:"team_id" binding:"required"`
	Name              string   `json:"name" binding:"required"`
	Type              string   `json:"type" binding:"required"`
	ManagerID         *uuid.UUID `json:"manager_id,omitempty"`
	PanelPCPs         []string `json:"panel_pcps,omitempty"`
	MaxTasksPerMember int      `json:"max_tasks_per_member,omitempty"`
	AutoAssign        bool     `json:"auto_assign"`
}

// CreateTeamMemberRequest represents the request body for creating a team member
type CreateTeamMemberRequest struct {
	MemberID     string      `json:"member_id" binding:"required"`
	UserID       string      `json:"user_id" binding:"required"`
	TeamID       uuid.UUID   `json:"team_id" binding:"required"`
	Name         string      `json:"name" binding:"required"`
	Role         string      `json:"role" binding:"required"`
	Email        string      `json:"email,omitempty"`
	Phone        string      `json:"phone,omitempty"`
	MaxTasks     int         `json:"max_tasks,omitempty"`
	Skills       []string    `json:"skills,omitempty"`
	Languages    []string    `json:"languages,omitempty"`
	SupervisorID *uuid.UUID  `json:"supervisor_id,omitempty"`
}

// TeamResponse wraps a team for API responses
type TeamResponse struct {
	Success bool   `json:"success"`
	Data    *Team  `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// TeamMemberResponse wraps a team member for API responses
type TeamMemberResponse struct {
	Success bool        `json:"success"`
	Data    *TeamMember `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
