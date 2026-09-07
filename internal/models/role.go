package models

import "strings"

type Role string

// stable role names, persisted and compared as strings
const (
	RoleHomeowner    Role = "Homeowner"
	RoleTradesperson Role = "Tradesperson"
	RoleAdmin        Role = "Administrator"
)

// Turns a free-form string into a Role, case-insensitively.
func ParseRole(s string) (Role, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "homeowner":
		return RoleHomeowner, true
	case "tradesperson":
		return RoleTradesperson, true
	case "administrator", "admin":
		return RoleAdmin, true
	default:
		return "", false
	}
}
