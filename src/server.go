package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	plaid "github.com/plaid/plaid-go/v40/plaid"
)

var (
	PLAID_CLIENT_ID                      = ""
	PLAID_SECRET                         = ""
	PLAID_ENV                            = ""
	PLAID_PRODUCTS                       = ""
	PLAID_COUNTRY_CODES                  = ""
	PLAID_REDIRECT_URI                   = ""
	APP_PORT                             = ""
	client              *plaid.APIClient = nil
)

var environments = map[string]plaid.Environment{
	"sandbox":    plaid.Sandbox,
	"production": plaid.Production,
}

var accessToken string
var userToken string
var itemID string

func init() {
	// Load env vars from .env file
	err := godotenv.Load()
	if err != nil {
		// Print a helpful message but continue - env vars may be provided by the environment
		fmt.Printf("Error when loading environment variables from .env file: %v\n", err)
	}

	// Set constants from env
	// Can be acquired from the Plaid Dashboard for free (https://dashboard.plaid.com/developers/keys)
	PLAID_CLIENT_ID = os.Getenv("PLAID_CLIENT_ID")
	PLAID_SECRET = os.Getenv("PLAID_SECRET")
	PLAID_ENV = os.Getenv("PLAID_ENV")
	PLAID_PRODUCTS = os.Getenv("PLAID_PRODUCTS")
	PLAID_COUNTRY_CODES = os.Getenv("PLAID_COUNTRY_CODES")
	PLAID_REDIRECT_URI = os.Getenv("PLAID_REDIRECT_URI")
	APP_PORT = os.Getenv("APP_PORT")

	// Initialize Plaid client
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", PLAID_CLIENT_ID)
	configuration.AddDefaultHeader("PLAID-SECRET", PLAID_SECRET)

	// Choose environment based on PLAID_ENV; default to sandbox
	env := strings.ToLower(PLAID_ENV)
	if envVal, ok := environments[env]; ok {
		configuration.UseEnvironment(envVal)
	} else {
		configuration.UseEnvironment(plaid.Sandbox)
	}
	client = plaid.NewAPIClient(configuration)
}

func main() {
	// Initialize Gin router
	r := gin.Default()

	r.GET("/api/transactions", transactions) // Endpoint to fetch transactions

	err := r.Run(":" + APP_PORT)
	if err != nil {
		panic("Unable to start server")
	}
}

// Check if the error is a Plaid error and renders it appropriately
func renderError(c *gin.Context, originalErr error) {
	if plaidError, err := plaid.ToPlaidError(originalErr); err == nil {
		// Return 200 and allow the front end to render the error.
		c.JSON(http.StatusOK, gin.H{"error": plaidError})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": originalErr.Error()})
}

func transactions(c *gin.Context) {
	ctx := context.Background()

	// Set cursor to empty to receive all historical updates
	var cursor *string

	// New transaction updates since "cursor"
	var added []plaid.Transaction
	var modified []plaid.Transaction
	var removed []plaid.RemovedTransaction // Removed transaction ids

	hasMore := true

	// Iterate through each page of new transaction updates for item
	for hasMore {
		request := plaid.NewTransactionsSyncRequest(accessToken)
		if cursor != nil {
			request.SetCursor(*cursor)
		}
		resp, _, err := client.PlaidApi.TransactionsSync(
			ctx,
		).TransactionsSyncRequest(*request).Execute()
		if err != nil {
			renderError(c, err)
			return
		}

		// Update cursor to the next cursor
		nextCursor := resp.GetNextCursor()
		cursor = &nextCursor

		if *cursor == "" {
			time.Sleep(2 * time.Second)
			continue
		}

		// Add this page of results
		added = append(added, resp.GetAdded()...)
		modified = append(modified, resp.GetModified()...)
		removed = append(removed, resp.GetRemoved()...)
		hasMore = resp.GetHasMore()
	}

	sort.Slice(added, func(i, j int) bool {
		return added[i].GetDate() < added[j].GetDate()
	})

	latestTransactions := added[len(added)-9:]
	fmt.Println(latestTransactions, "LATEST TRANSACTIONS") // Debug print

	// Return the latest transactions as JSON
	c.JSON(http.StatusOK, gin.H{
		"latest_transactions": latestTransactions,
	})
}
