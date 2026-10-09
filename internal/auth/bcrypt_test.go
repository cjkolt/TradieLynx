package auth

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	password := "a test-only password with spaces"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == password {
		t.Fatal("stored hash must not equal plaintext password")
	}
	if err := CompareHashAndPassword(hash, password); err != nil {
		t.Fatalf("correct password failed verification: %v", err)
	}
	if err := CompareHashAndPassword(hash, "wrong password"); err == nil {
		t.Fatal("incorrect password unexpectedly passed verification")
	}
}

func TestMalformedPasswordHashIsRejected(t *testing.T) {
	if err := CompareHashAndPassword("not-a-valid-bcrypt-hash", "password"); err == nil {
		t.Fatal("malformed bcrypt hash unexpectedly passed verification")
	}
}
