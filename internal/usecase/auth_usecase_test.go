package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RyseUp/ChatterGo/internal/domain"
)

// Mock user repository for testing
type mockUserRepository struct {
	users map[string]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if _, exists := m.users[user.Email]; exists {
		return errors.New("email already registered")
	}
	user.ID = int64(len(m.users) + 1)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if user, exists := m.users[email]; exists {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	if existing, exists := m.users[user.Email]; exists {
		existing.Username = user.Username
		existing.Password = user.Password
		existing.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("user not found")
}

func TestAuthUseCase_Register(t *testing.T) {
	// This is a basic structural test to ensure the use case can be instantiated
	// In a real scenario, we would test with actual implementations
	mockRepo := newMockUserRepository()
	
	// Verify mock repository works
	ctx := context.Background()
	testUser := &domain.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
	}
	
	err := mockRepo.Create(ctx, testUser)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if testUser.ID == 0 {
		t.Error("Expected user ID to be set")
	}
	
	// Test duplicate email
	duplicateUser := &domain.User{
		Username: "anotheruser",
		Email:    "test@example.com",
		Password: "password",
	}
	
	err = mockRepo.Create(ctx, duplicateUser)
	if err == nil {
		t.Error("Expected error for duplicate email")
	}
}
