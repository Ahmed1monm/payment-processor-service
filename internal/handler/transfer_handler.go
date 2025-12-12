package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	"paytabs/internal/errors"
	"paytabs/internal/service"
)

// TransferHandler handles transfer endpoints.
type TransferHandler struct {
	transferService service.TransferService
}

// NewTransferHandler creates a new transfer handler.
func NewTransferHandler(transferService service.TransferService) *TransferHandler {
	return &TransferHandler{transferService: transferService}
}

// TransferRequest represents a transfer request.
type TransferRequest struct {
	SourceAccountID      string `json:"source_account_id" validate:"required,uuid"`
	DestinationAccountID string `json:"destination_account_id" validate:"required,uuid"`
	Amount               string `json:"amount" validate:"required"`
}

// TransferResponse represents a transfer response.
type TransferResponse struct {
	TransferID string `json:"transfer_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// ProcessTransfer godoc
// @Summary Process an account-to-account transfer
// @Tags transfers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body TransferRequest true "Transfer data"
// @Success 200 {object} TransferResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /transfers [post]
func (h *TransferHandler) ProcessTransfer(c echo.Context) error {
	var req TransferRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid request body",
			Code:  "INVALID_REQUEST",
		})
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: err.Error(),
			Code:  "VALIDATION_ERROR",
		})
	}

	// Parse account IDs
	sourceAccountID, err := uuid.Parse(req.SourceAccountID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid source_account_id",
			Code:  "INVALID_UUID",
		})
	}

	destinationAccountID, err := uuid.Parse(req.DestinationAccountID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid destination_account_id",
			Code:  "INVALID_UUID",
		})
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid amount",
			Code:  "INVALID_AMOUNT",
		})
	}

	// Process transfer
	transfer, err := h.transferService.ProcessTransfer(
		c.Request().Context(),
		sourceAccountID,
		destinationAccountID,
		amount,
	)

	if err != nil {
		httpErr := errors.MapErrorToHTTP(err)
		return echo.NewHTTPError(httpErr.StatusCode, httpErr.ToErrorResponse())
	}

	status := "completed"
	message := "Transfer completed successfully"
	if transfer.Status == "failed" {
		status = "failed"
		message = transfer.ErrorMessage
		if message == "" {
			message = "Transfer processing failed"
		}
	}

	return c.JSON(http.StatusOK, TransferResponse{
		TransferID: transfer.ID.String(),
		Status:      status,
		Message:     message,
	})
}

