package models

type Data struct {
	Title string
	User  LoggedUser
}

// Request identity added by auth middleware.
type Claims struct {
	UserID   int64
	Role     Role
	IsActive bool
}
