package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"paytabs/internal/errors"
	"paytabs/internal/service"
)

// AccountHandler handles account endpoints.
type AccountHandler struct {
	accountService service.AccountService
}

// NewAccountHandler creates a new account handler.
func NewAccountHandler(accountService service.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}

// BalanceResponse represents an account balance response.
type BalanceResponse struct {
	AccountID uuid.UUID `json:"account_id"`
	Balance   string    `json:"balance"`
}

// GetBalance godoc
// @Summary Get account balance
// @Tags accounts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Account ID"
// @Success 200 {object} BalanceResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /accounts/{id}/balance [get]
func (h *AccountHandler) GetBalance(c echo.Context) error {
	accountIDStr := c.Param("id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid account ID",
			Code:  "INVALID_UUID",
		})
	}

	balance, err := h.accountService.GetBalance(c.Request().Context(), accountID)
	if err != nil {
		httpErr := errors.MapErrorToHTTP(err)
		return echo.NewHTTPError(httpErr.StatusCode, httpErr.ToErrorResponse())
	}

	return c.JSON(http.StatusOK, BalanceResponse{
		AccountID: accountID,
		Balance:   balance.String(),
	})
}
