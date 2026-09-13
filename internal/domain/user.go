package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleCustomer Role = "CUSTOMER"
	RoleStaff    Role = "STAFF"
	RoleAdmin    Role = "ADMIN"
)

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	Phone        string
	PasswordHash string
	Role         Role
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) DisplayName() string {
	// Returns "Heet C." format for reviews
	if len(u.Name) == 0 {
		return "Unknown"
	}
	parts := splitName(u.Name)
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + " " + string(parts[len(parts)-1][0]) + "."
}

func splitName(name string) []string {
	var parts []string
	word := ""
	for _, c := range name {
		if c == ' ' {
			if word != "" {
				parts = append(parts, word)
				word = ""
			}
		} else {
			word += string(c)
		}
	}
	if word != "" {
		parts = append(parts, word)
	}
	return parts
}
