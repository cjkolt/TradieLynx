package models

type User struct { // Structure for user registration
	ID             int64
	FirstName      string
	LastName       string
	Email          string
	HashedPassword string
	Role           Role
	IsActive       bool
}

type AuthUser struct { // Structure for verifying login
	ID        int64
	FirstName string
	Email     string
	Role      string
	IsActive  bool
}

type LoggedUser struct { // Structure for logged-in user info for custom pages
	ID        int64
	FirstName string
	Role      Role
	IsActive  bool
}
