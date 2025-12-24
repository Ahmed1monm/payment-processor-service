package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"paytabs/internal/auth"
	"paytabs/internal/model"
	"paytabs/internal/repository"
)

// MockAccountRepository is a mock implementation of AccountRepository.
type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) Create(ctx context.Context, account *model.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountRepository) Update(ctx context.Context, account *model.Account) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockAccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountRepository) FindByEmail(ctx context.Context, email string) (*model.Account, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountRepository) ListActive(ctx context.Context) ([]model.Account, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Account), args.Error(1)
}

func (m *MockAccountRepository) FindByIDOrCreate(ctx context.Context, account *model.Account) (*model.Account, error) {
	args := m.Called(ctx, account)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context, repo repository.AccountRepository) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockAccountRepository) FindByIDForUpdateTx(ctx context.Context, tx interface{}, id uuid.UUID) (*model.Account, error) {
	args := m.Called(ctx, tx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
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
		isMerchant    bool
		setupMock     func(*MockAccountRepository)
		expectedError error
	}{
		{
			name:       "successful registration",
			email:      "test@example.com",
			password:   "password123",
			nameField:  "Test User",
			isMerchant: false,
			setupMock: func(m *MockAccountRepository) {
				m.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, gorm.ErrRecordNotFound)
				m.On("Create", mock.Anything, mock.AnythingOfType("*model.Account")).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:       "account already exists",
			email:      "existing@example.com",
			password:   "password123",
			nameField:  "Existing User",
			isMerchant: false,
			setupMock: func(m *MockAccountRepository) {
				m.On("FindByEmail", mock.Anything, "existing@example.com").Return(&model.Account{Email: "existing@example.com"}, nil)
			},
			expectedError: ErrUserAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAccountRepository)
			tt.setupMock(mockRepo)

			jwtService := auth.NewJWTService("test-secret")
			mockTokenStore := new(MockTokenStore)

			service := NewAuthService(mockRepo, jwtService, mockTokenStore)
			account, err := service.Register(context.Background(), tt.email, tt.password, tt.nameField, tt.isMerchant)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, account)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, account)
				assert.Equal(t, tt.email, account.Email)
				assert.Equal(t, tt.nameField, account.Name)
				assert.NotEmpty(t, account.PasswordHash)
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
		setupMock     func(*MockAccountRepository, *MockTokenStore)
		expectedError error
	}{
		{
			name:     "successful login",
			email:    "test@example.com",
			password: "password123",
			setupMock: func(mRepo *MockAccountRepository, mToken *MockTokenStore) {
				// Generate a real bcrypt hash for the password
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), 10)
				accountID := uuid.New()
				mRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&model.Account{
					ID:           accountID,
					Email:        "test@example.com",
					PasswordHash: string(hashedPassword),
				}, nil)
				// Convert UUID to uint for token store (using first 4 bytes)
				accountIDUint := uint(accountID[0]) + uint(accountID[1])<<8 + uint(accountID[2])<<16 + uint(accountID[3])<<24
				mToken.On("StoreRefreshToken", mock.Anything, mock.Anything, accountIDUint, "test@example.com", mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:     "invalid credentials - account not found",
			email:    "notfound@example.com",
			password: "password123",
			setupMock: func(mRepo *MockAccountRepository, mToken *MockTokenStore) {
				mRepo.On("FindByEmail", mock.Anything, "notfound@example.com").Return(nil, gorm.ErrRecordNotFound)
			},
			expectedError: ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAccountRepository)
			mockTokenStore := new(MockTokenStore)
			tt.setupMock(mockRepo, mockTokenStore)

			jwtService := auth.NewJWTService("test-secret")
			service := NewAuthService(mockRepo, jwtService, mockTokenStore)

			accessToken, refreshToken, account, err := service.Login(context.Background(), tt.email, tt.password)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
				assert.Nil(t, account)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, accessToken)
				assert.NotEmpty(t, refreshToken)
				assert.NotNil(t, account)
				assert.Equal(t, tt.email, account.Email)
			}

			mockRepo.AssertExpectations(t)
			mockTokenStore.AssertExpectations(t)
		})
	}
}
