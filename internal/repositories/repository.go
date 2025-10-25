package repositories

import (
	"context"
	"time"

	"github.com/RyseUp/ChatterGo/internal/models"
)

type Repository interface {
	Ping() error
	Transaction(ctx context.Context, txFunc func(Repository) error) error
	UserRepository
	ConversationRepository
	MessageRepository
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUserID(ctx context.Context, id uint) (*models.User, error)
	GetUserByUserEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, id uint, updates map[string]interface{}) error
	UpdateUserRefreshToken(ctx context.Context, id uint, refreshToken *string) error
	UpdateUserLastLogin(ctx context.Context, id uint, lastLogin time.Time) error
}

type ConversationRepository interface {
	CreateConversation(ctx context.Context, conversation *models.Conversation) error
	GetConversationByID(ctx context.Context, id uint) (*models.Conversation, error)
	GetConversationsByUserID(ctx context.Context, userID uint, limit, offset int) ([]models.Conversation, int64, error)
	GetDirectConversationBetweenUsers(ctx context.Context, userID1, userID2 uint) (*models.Conversation, error)
	UpdateConversation(ctx context.Context, id uint, updates map[string]interface{}) error
	DeleteConversation(ctx context.Context, id uint) error

	// ConversationMember methods
	AddConversationMembers(ctx context.Context, members []*models.ConversationMember) error
	RemoveConversationMember(ctx context.Context, conversationID, userID uint) error
	GetConversationMembers(ctx context.Context, conversationID uint) ([]models.ConversationMember, error)
	IsConversationMember(ctx context.Context, conversationID, userID uint) (bool, error)
	GetMemberRole(ctx context.Context, conversationID, userID uint) (*models.MemberRole, error)
}

type MessageRepository interface {
	CreateMessage(ctx context.Context, message *models.Message) error
	GetMessageByID(ctx context.Context, id uint) (*models.Message, error)
	GetMessagesByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]models.Message, int64, error)
	UpdateMessage(ctx context.Context, id uint, updates map[string]interface{}) error
	DeleteMessage(ctx context.Context, id uint) error
}
