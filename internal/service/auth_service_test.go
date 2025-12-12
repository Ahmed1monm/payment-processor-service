package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"paytabs/internal/auth"
	"paytabs/internal/model"
)

// MockUserRepository is a mock implementation of UserRepository.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context) ([]model.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.User), args.Error(1)
}

// MockTokenStore is a mock implementation of TokenStoreInterface.
type MockTokenStore struct {
	mock.Mock
}

func (m *MockTokenStore) StoreRefreshToken(ctx context.Context, tokenID string, userID uint, email string, ttl time.Duration) error {
	args := m.Called(ctx, tokenID, userID, email, ttl)
	return args.Error(0)
}

func (m *MockTokenStore) GetRefreshToken(ctx context.Context, tokenID string) (uint, string, error) {
	args := m.Called(ctx, tokenID)
	return args.Get(0).(uint), args.String(1), args.Error(2)
}

func (m *MockTokenStore) DeleteRefreshToken(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockTokenStore) BlacklistAccessToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	args := m.Called(ctx, tokenID, ttl)
	return args.Error(0)
}

func (m *MockTokenStore) IsAccessTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	args := m.Called(ctx, tokenID)
	return args.Bool(0), args.Error(1)
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		nameField     string
		setupMock     func(*MockUserRepository)
		expectedError error
	}{
		{
			name:      "successful registration",
			email:     "test@example.com",
			password:  "password123",
			nameField: "Test User",
			setupMock: func(m *MockUserRepository) {
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, gorm.ErrRecordNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "user already exists",
			email:     "existing@example.com",
			password:  "password123",
			nameField: "Existing User",
			setupMock: func(m *MockUserRepository) {
				m.On("FindByEmail", mock.Anything, "existing@example.com").Return(&model.User{Email: "existing@example.com"}, nil)
			},
			expectedError: ErrUserAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			tt.setupMock(mockRepo)

			jwtService := auth.NewJWTService("test-secret")
			mockTokenStore := new(MockTokenStore)

			service := NewAuthService(mockRepo, jwtService, mockTokenStore)
			user, err := service.Register(context.Background(), tt.email, tt.password, tt.nameField)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
				assert.Equal(t, tt.nameField, user.Name)
				assert.NotEmpty(t, user.PasswordHash)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		setupMock     func(*MockUserRepository, *MockTokenStore)
		expectedError error
	}{
		{
			name:     "successful login",
			email:    "test@example.com",
			password: "password123",
			setupMock: func(mRepo *MockUserRepository, mToken *MockTokenStore) {
				// Generate a real bcrypt hash for the password
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
				mRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&model.User{
					ID:           1,
					Email:        "test@example.com",
					PasswordHash: string(hashedPassword),
				}, nil)
				mToken.On("StoreRefreshToken", mock.Anything, mock.Anything, uint(1), "test@example.com", mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:     "invalid credentials - user not found",
			email:    "notfound@example.com",
			password: "password123",
			setupMock: func(mRepo *MockUserRepository, mToken *MockTokenStore) {
				mRepo.On("FindByEmail", mock.Anything, "notfound@example.com").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockUserRepository)
			mockTokenStore := new(MockTokenStore)
			tt.setupMock(mockRepo, mockTokenStore)

			jwtService := auth.NewJWTService("test-secret")
			service := NewAuthService(mockRepo, jwtService, mockTokenStore)

			accessToken, refreshToken, user, err := service.Login(context.Background(), tt.email, tt.password)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, accessToken)
				assert.NotEmpty(t, refreshToken)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
			}

			mockRepo.AssertExpectations(t)
			mockTokenStore.AssertExpectations(t)
		})
	}
}
