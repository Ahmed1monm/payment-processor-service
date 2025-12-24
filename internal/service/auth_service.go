package service

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"paytabs/internal/auth"
	"paytabs/internal/model"
	"paytabs/internal/repository"
	"github.com/google/uuid"
)

const bcryptCost = 10

var (
	// ErrInvalidCredentials is returned when email or password is incorrect.
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrUserAlreadyExists is returned when trying to register an existing user.
	ErrUserAlreadyExists = errors.New("user already exists")
	// ErrInvalidRefreshToken is returned when refresh token is invalid or expired.
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

// AuthService handles authentication operations.
type AuthService interface {
	Register(ctx context.Context, email, password, name string, isMerchant bool) (*model.Account, error)
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, account *model.Account, err error)
	RefreshToken(ctx context.Context, refreshToken string) (accessToken string, err error)
	Logout(ctx context.Context, refreshToken string) error
}

type authService struct {
	accountRepo repository.AccountRepository
	jwtService  *auth.JWTService
	tokenStore   auth.TokenStoreInterface
}

// NewAuthService creates a new authentication service.
func NewAuthService(accountRepo repository.AccountRepository, jwtService *auth.JWTService, tokenStore auth.TokenStoreInterface) AuthService {
	return &authService{
		accountRepo: accountRepo,
		jwtService:  jwtService,
		tokenStore:  tokenStore,
	}
}

// Register creates a new account with hashed password.
func (s *authService) Register(ctx context.Context, email, password, name string, isMerchant bool) (*model.Account, error) {
	// Check if account already exists
	existing, err := s.accountRepo.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, ErrUserAlreadyExists
	}
	// If error is not "record not found", return it (could be a database error)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("check account existence: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Create account
	account := &model.Account{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Name:         name,
		IsMerchant:   isMerchant,
		Active:       true,
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	return account, nil
}

// Login authenticates an account and returns access and refresh tokens.
func (s *authService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, account *model.Account, err error) {
	// Find account by email
	account, err = s.accountRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	// Generate access token (using account ID as uint)
	accountIDUint := uint(account.ID[0]) + uint(account.ID[1])<<8 + uint(account.ID[2])<<16 + uint(account.ID[3])<<24
	accessToken, err = s.jwtService.GenerateAccessToken(accountIDUint, account.Email)
	if err != nil {
		return "", "", nil, fmt.Errorf("generate access token: %w", err)
	}

	// Generate refresh token
	tokenID, refreshToken, err := s.jwtService.GenerateRefreshToken(accountIDUint, account.Email)
	if err != nil {
		return "", "", nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Store refresh token in Redis
	if err := s.tokenStore.StoreRefreshToken(ctx, tokenID, accountIDUint, account.Email, auth.RefreshTokenExpiry); err != nil {
		return "", "", nil, fmt.Errorf("store refresh token: %w", err)
	}

	return accessToken, refreshToken, account, nil
}

// RefreshToken validates a refresh token and returns a new access token.
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (accessToken string, err error) {
	// Validate refresh token
	claims, err := s.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return "", ErrInvalidRefreshToken
	}

	// Extract token ID
	tokenID, err := s.jwtService.ExtractTokenID(refreshToken)
	if err != nil {
		return "", ErrInvalidRefreshToken
	}

	// Verify token exists in Redis
	storedUserID, storedEmail, err := s.tokenStore.GetRefreshToken(ctx, tokenID)
	if err != nil {
		return "", ErrInvalidRefreshToken
	}

	// Verify token matches stored data
	if storedUserID != claims.UserID || storedEmail != claims.Email {
		return "", ErrInvalidRefreshToken
	}

	// Generate new access token
	accessToken, err = s.jwtService.GenerateAccessToken(claims.UserID, claims.Email)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}

	return accessToken, nil
}

// Logout invalidates a refresh token.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	// Extract token ID
	tokenID, err := s.jwtService.ExtractTokenID(refreshToken)
	if err != nil {
		return ErrInvalidRefreshToken
	}

	// Delete refresh token from Redis
	return s.tokenStore.DeleteRefreshToken(ctx, tokenID)
}
