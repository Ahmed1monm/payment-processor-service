package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"paytabs/internal/errors"
	"paytabs/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRequest represents a user registration request.
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
}

// LoginRequest represents a user login request.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest represents a logout request.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// AuthResponse represents an authentication response.
type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token,omitempty"`
	User         interface{} `json:"user,omitempty"`
}

// Register godoc
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.authService.Register(c.Request().Context(), req.Email, req.Password, req.Name)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			return echo.NewHTTPError(http.StatusConflict, errors.ErrorResponse{
				Error: err.Error(),
				Code:  "USER_ALREADY_EXISTS",
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, errors.ErrorResponse{
			Error: "failed to register user",
			Code:  "REGISTRATION_FAILED",
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "user registered successfully",
		"user":    user,
	})
}

// Login godoc
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	accessToken, refreshToken, user, err := h.authService.Login(c.Request().Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			return echo.NewHTTPError(http.StatusUnauthorized, errors.ErrorResponse{
				Error: err.Error(),
				Code:  "INVALID_CREDENTIALS",
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, errors.ErrorResponse{
			Error: "failed to login",
			Code:  "LOGIN_FAILED",
		})
	}

	return c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

// Refresh godoc
// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req RefreshRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	accessToken, err := h.authService.RefreshToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		if err == service.ErrInvalidRefreshToken {
			return echo.NewHTTPError(http.StatusUnauthorized, errors.ErrorResponse{
				Error: err.Error(),
				Code:  "INVALID_REFRESH_TOKEN",
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, errors.ErrorResponse{
			Error: "failed to refresh token",
			Code:  "REFRESH_FAILED",
		})
	}

	return c.JSON(http.StatusOK, AuthResponse{
		AccessToken: accessToken,
	})
}

// Logout godoc
// @Summary Logout user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "Refresh token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	var req LogoutRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.authService.Logout(c.Request().Context(), req.RefreshToken); err != nil {
		if err == service.ErrInvalidRefreshToken {
			return echo.NewHTTPError(http.StatusUnauthorized, errors.ErrorResponse{
				Error: err.Error(),
				Code:  "INVALID_REFRESH_TOKEN",
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError, errors.ErrorResponse{
			Error: "failed to logout",
			Code:  "LOGOUT_FAILED",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// Helper function to handle GORM errors
func handleDBError(err error) *echo.HTTPError {
	if err == gorm.ErrRecordNotFound {
		return echo.NewHTTPError(http.StatusNotFound, errors.ErrorResponse{
			Error: "record not found",
			Code:  "NOT_FOUND",
		})
	}
	return echo.NewHTTPError(http.StatusInternalServerError, errors.ErrorResponse{
		Error: "database error",
		Code:  "DATABASE_ERROR",
	})
}

