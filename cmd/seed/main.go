package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"paytabs/internal/config"
	"paytabs/internal/db"
	"paytabs/internal/model"
	"paytabs/internal/repository"
)

const accountsAPIURL = "https://gist.githubusercontent.com/paytabscom/b590d72ae115226e288a9c8a15ba2888/raw/ac0d615060b02e755c94116e4e5a5af530bc4bb1/accounts.json"

// SeedAccountData represents the structure from the external API.
type SeedAccountData struct {
	ID      string `json:"id"`
	Active  bool   `json:"active"`
	Name    string `json:"name"`
	Balance string `json:"balance"`
}

func main() {
	log.Println("Starting seed script...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	gormDB, err := db.NewMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Connected to database")

	// Run migrations to ensure schema is up to date
	if err := gormDB.AutoMigrate(&model.Account{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Fetch accounts from API
	log.Printf("Fetching accounts from: %s", accountsAPIURL)
	accounts, err := fetchAccountsFromAPI(accountsAPIURL)
	if err != nil {
		log.Fatalf("Failed to fetch accounts: %v", err)
	}
	log.Printf("Fetched %d accounts from API", len(accounts))

	// Convert to model.Account
	modelAccounts := make([]model.Account, 0, len(accounts))
	skipped := 0
	for _, item := range accounts {
		accountID, err := uuid.Parse(item.ID)
		if err != nil {
			log.Printf("Skipping account with invalid UUID: %s", item.ID)
			skipped++
			continue
		}

		balance, err := decimal.NewFromString(item.Balance)
		if err != nil {
			log.Printf("Skipping account %s with invalid balance: %s", item.ID, item.Balance)
			skipped++
			continue
		}

		account := model.Account{
			ID:      accountID,
			Name:    item.Name,
			Balance: balance,
			Active:  item.Active,
		}
		modelAccounts = append(modelAccounts, account)
	}

	if skipped > 0 {
		log.Printf("Skipped %d invalid accounts", skipped)
	}

	// Seed accounts into database
	accountRepo := repository.NewAccountRepository(gormDB)
	ctx := context.Background()

	log.Println("Seeding accounts into database...")
	seeded, updated, err := seedAccounts(ctx, accountRepo, modelAccounts)
	if err != nil {
		log.Fatalf("Failed to seed accounts: %v", err)
	}

	log.Printf("Seed completed successfully!")
	log.Printf("  - New accounts created: %d", seeded)
	log.Printf("  - Existing accounts updated: %d", updated)
	log.Printf("  - Total accounts processed: %d", seeded+updated)
}

// fetchAccountsFromAPI fetches account data from the external API.
func fetchAccountsFromAPI(url string) ([]SeedAccountData, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var accounts []SeedAccountData
	if err := json.Unmarshal(body, &accounts); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return accounts, nil
}

// seedAccounts seeds accounts into the database, creating new ones or updating existing ones.
func seedAccounts(ctx context.Context, repo repository.AccountRepository, accounts []model.Account) (seeded int, updated int, err error) {
	for _, account := range accounts {
		existing, err := repo.FindByID(ctx, account.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			return seeded, updated, fmt.Errorf("error checking account %s: %w", account.ID, err)
		}

		if existing != nil {
			// Update existing account
			existing.Name = account.Name
			existing.Balance = account.Balance
			existing.Active = account.Active
			if err := repo.Update(ctx, existing); err != nil {
				return seeded, updated, fmt.Errorf("error updating account %s: %w", account.ID, err)
			}
			updated++
		} else {
			// Create new account
			if err := repo.Create(ctx, &account); err != nil {
				return seeded, updated, fmt.Errorf("error creating account %s: %w", account.ID, err)
			}
			seeded++
		}
	}

	return seeded, updated, nil
}
