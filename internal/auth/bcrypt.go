package auth

import "golang.org/x/crypto/bcrypt"

// Used during registration
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Used when db attempts to retrieve a user (login), bcrypt func wrapper
func CompareHashAndPassword(passwordHash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
}
