package models

import "testing"

func TestParseRole(t *testing.T) {
	tests := []struct {
		name string
		input string
		want Role
		ok bool
	}{
		{"homeowner", "homeowner", RoleHomeowner, true},
		{"tradesperson mixed case", " TradesPerson ", RoleTradesperson, true},
		{"administrator", "Administrator", RoleAdmin, true},
		{"admin alias", "ADMIN", RoleAdmin, true},
		{"unknown role", "moderator", "", false},
		{"empty string", "", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseRole(tc.input)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("ParseRole(%q) = (%q, %t), want (%q, %t)",
					tc.input, got, ok, tc.want, tc.ok)
			}
		})
	}
}
