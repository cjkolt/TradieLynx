package models

type User struct {
	ID             int64
	FirstName      string
	LastName       string
	Email          string
	HashedPassword string
	Role           Role
	IsActive       bool
}

type LoggedUser struct {
	ID        int64
	FirstName string
	Role      Role
	IsActive  bool
}
