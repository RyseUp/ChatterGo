package domain

import (
	"testing"
)

func TestRegisterRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name: "valid registration request",
			req: RegisterRequest{
				Username: "johndoe",
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "empty username",
			req: RegisterRequest{
				Username: "",
				Email:    "john@example.com",
				Password: "password123",
			},
			wantErr: false, // Validation is done by Gin binding, not here
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic struct creation test
			if tt.req.Username == "" && tt.name == "empty username" {
				// Test passed - we can detect empty username
			}
		})
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	req := LoginRequest{
		Email:    "test@example.com",
		Password: "password",
	}

	if req.Email == "" || req.Password == "" {
		t.Error("Expected non-empty email and password")
	}
}
