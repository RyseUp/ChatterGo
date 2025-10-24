package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	
	DefaultCost = bcrypt.DefaultCost
)


func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}


func CheckPassword(password, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return fmt.Errorf("password does not match: %w", err)
	}
	return nil
}
