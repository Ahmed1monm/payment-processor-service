package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"paytabs/internal/model"
	"paytabs/internal/service"
)

// SeedHandler handles seed data endpoints.
type SeedHandler struct {
	accountService service.AccountService
}

// NewSeedHandler creates a new seed handler.
func NewSeedHandler(accountService service.AccountService) *SeedHandler {
	return &SeedHandler{accountService: accountService}
}

// SeedAccountsRequest represents the structure from the external API.
type SeedAccountsRequest struct {
	ID      string `json:"id"`
	Active  bool   `json:"active"`
	Name    string `json:"name"`
	Balance string `json:"balance"`
}

// SeedAccountsResponse represents the seed response.
type SeedAccountsResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// SeedAccounts godoc
// @Summary Seed accounts from external API
// @Tags seed
// @Produce json
// @Success 200 {object} SeedAccountsResponse
// @Failure 500 {object} map[string]string
// @Router /seed/accounts [get]
func (h *SeedHandler) SeedAccounts(c echo.Context) error {
	// Fetch accounts from external API
	url := "https://gist.githubusercontent.com/paytabscom/b590d72ae115226e288a9c8a15ba2888/raw/ac0d615060b02e755c94116e4e5a5af530bc4bb1/accounts.json"
	resp, err := http.Get(url)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to fetch accounts: %v", err),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("external API returned status: %d", resp.StatusCode),
		})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to read response: %v", err),
		})
	}

	// Parse JSON response
	var seedData []SeedAccountsRequest
	if err := json.Unmarshal(body, &seedData); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to parse JSON: %v", err),
		})
	}

	// Convert to model.Account
	accounts := make([]model.Account, 0, len(seedData))
	for _, item := range seedData {
		accountID, err := uuid.Parse(item.ID)
		if err != nil {
			// Skip invalid UUIDs
			continue
		}

		account := model.Account{
			ID:        accountID,
			Name:      item.Name,
			Email:     fmt.Sprintf("account-%s@example.com", accountID.String()), // Generate email for seeded accounts
			Active:    item.Active,
			IsMerchant: false, // Default to non-merchant for seeded accounts
		}
		accounts = append(accounts, account)
	}

	// Seed accounts
	count, err := h.accountService.SeedAccounts(c.Request().Context(), accounts)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to seed accounts: %v", err),
		})
	}

	return c.JSON(http.StatusOK, SeedAccountsResponse{
		Message: "Accounts seeded successfully",
		Count:   count,
	})
}

