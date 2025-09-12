package models

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleMaster UserRole = "master"
)

func (r UserRole) String() string {
	return string(r)
}

func (r UserRole) IsValid() bool {
	return r == UserRoleAdmin || r == UserRoleMaster
}

type User struct {
	ID         uint           `json:"id" gorm:"primarykey"`
	TelegramID int64          `json:"telegram_id" gorm:"uniqueIndex;not null"`
	Name       string         `json:"name" gorm:"size:100;not null"`
	Username   *string        `json:"username"`
	FirstName  *string        `json:"first_name"`
	LastName   *string        `json:"last_name"`
	Role       UserRole       `json:"role" gorm:"type:varchar(20);not null"`
	Language   string         `json:"language" gorm:"size:5;default:'ru'"`
	IsActive   bool           `json:"is_active" gorm:"default:true"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Связи
	Devices []Device `gorm:"foreignKey:MasterID" json:"devices,omitempty"`
}

func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

func (u *User) IsMaster() bool {
	return u.Role == UserRoleMaster
}

func (u *User) FullName() string {
	name := ""
	if u.FirstName != nil {
		name += *u.FirstName
	}
	if u.LastName != nil {
		if name != "" {
			name += " "
		}
		name += *u.LastName
	}
	if name == "" && u.Username != nil {
		name = *u.Username
	}
	if name == "" {
		name = u.Name
	}
	return name
}

func (u *User) TableName() string {
	return "users"
}

func (u *User) GetLanguage() string {
	if u.Language == "" {
		return "ru"
	}
	return u.Language
}

func (u *User) SetLanguage(lang string) {
	u.Language = lang
}

func (u *User) IsValidLanguage(lang string) bool {
	return lang == "ru" || lang == "uz" || lang == "en"
}
