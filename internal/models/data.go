package models

type Data struct { // Used by base.go for filling w/ content
	Title string
	User  LoggedUser
}

// Represents the authenticated user for this request.
// Fields are meant to be read-only after middleware attaches them.
type Claims struct {
	UserID   int64 // DB id of the user
	Role     Role  // role at the time if request (from session/DB)
	IsActive bool
}
