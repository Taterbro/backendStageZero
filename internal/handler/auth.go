package handler

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/Taterbro/backendStageZero/internal/model"
	"github.com/Taterbro/backendStageZero/internal/service"
	"github.com/Taterbro/backendStageZero/internal/utils"
)

// In-memory store for invalidated refresh tokens (use database in production)
var (
	invalidatedTokensMutex sync.RWMutex
	invalidatedTokens      = make(map[string]bool)
)

// InvalidateRefreshToken marks a refresh token as invalid
func InvalidateRefreshToken(token string) {
	invalidatedTokensMutex.Lock()
	defer invalidatedTokensMutex.Unlock()
	invalidatedTokens[token] = true
}

// IsTokenInvalidated checks if a refresh token has been invalidated
func IsTokenInvalidated(token string) bool {
	invalidatedTokensMutex.RLock()
	defer invalidatedTokensMutex.RUnlock()
	return invalidatedTokens[token]
}

// GitHubOAuthHandler redirects user to GitHub OAuth authorization
func GitHubOAuthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJson(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Status:  "error",
			Message: "Method not allowed",
		})
		return
	}

	// Generate PKCE challenge
	_, err := service.GeneratePKCEChallenge()
	if err != nil {
		log.Printf("Failed to generate PKCE challenge: %v", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to generate authentication challenge",
		})
		return
	}

	// Store the PKCE verifier in session (use Redis/database in production)
	// For now, we're passing state which includes the verifier
	stateToken, err := service.GenerateStateToken()
	if err != nil {
		log.Printf("Failed to generate state token: %v", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to generate state token",
		})
		return
	}

	// In production, store: stateToken -> pkceChallenge.Verifier mapping
	// For now, using state as the challenge (simplified for demo)

	authURL := service.GetGitHubAuthURL(stateToken)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// GitHubCallbackHandler handles the OAuth callback from GitHub
func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJson(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Status:  "error",
			Message: "Method not allowed",
		})
		return
	}

	// Get authorization code and state
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errParam := r.URL.Query().Get("error")

	// Check for OAuth errors
	if errParam != "" {
		errDescription := r.URL.Query().Get("error_description")
		log.Printf("OAuth error: %s - %s", errParam, errDescription)
		utils.WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("OAuth error: %s", errParam),
		})
		return
	}

	// Validate code and state
	if code == "" || state == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Missing authorization code or state",
		})
		return
	}

	// Exchange code for access token
	accessToken, err := service.ExchangeCodeForToken(code)
	if err != nil {
		log.Printf("Failed to exchange code for token: %v", err)
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to exchange authorization code",
		})
		return
	}

	// Get GitHub user information
	githubUser, err := service.GetGitHubUser(accessToken)
	if err != nil {
		log.Printf("Failed to get GitHub user: %v", err)
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to fetch user information",
		})
		return
	}

	// In production, create or retrieve user from database
	userID := fmt.Sprintf("github_%d", githubUser.ID)

	// Generate token pair
	tokenPair, err := service.GenerateTokenPair(userID)
	if err != nil {
		log.Printf("Failed to generate token pair: %v", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to generate authentication tokens",
		})
		return
	}

	// In production, store the refresh token in database with user ID and expiration
	// Also store the mapping of state token to PKCE verifier for verification

	// Return token response
	utils.WriteJson(w, http.StatusOK, model.TokenResponse{
		Status:       "success",
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	})
}

// RefreshTokenHandler handles token refresh requests
func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJson(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Status:  "error",
			Message: "Method not allowed",
		})
		return
	}

	var req model.RefreshTokenRequest
	if err := utils.ParseJson(r, &req); err != nil {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Invalid request body",
		})
		return
	}

	// Validate refresh token is not empty
	if req.RefreshToken == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Refresh token is required",
		})
		return
	}

	// Check if token has been invalidated
	if IsTokenInvalidated(req.RefreshToken) {
		log.Printf("Attempted to use invalidated refresh token")
		utils.WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: "Refresh token has been invalidated",
		})
		return
	}

	// In production: validate token signature, expiration, and existence in database
	userID, err := service.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		log.Printf("Invalid refresh token: %v", err)
		utils.WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: "Invalid refresh token",
		})
		return
	}

	// Invalidate old refresh token immediately
	InvalidateRefreshToken(req.RefreshToken)

	// Generate new token pair
	newTokenPair, err := service.GenerateTokenPair(userID)
	if err != nil {
		log.Printf("Failed to generate new token pair: %v", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Failed to generate new tokens",
		})
		return
	}

	// Return new token pair
	utils.WriteJson(w, http.StatusOK, model.TokenResponse{
		Status:       "success",
		AccessToken:  newTokenPair.AccessToken,
		RefreshToken: newTokenPair.RefreshToken,
	})
}

// LogoutHandler invalidates the refresh token
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJson(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Status:  "error",
			Message: "Method not allowed",
		})
		return
	}

	var req model.LogoutRequest
	if err := utils.ParseJson(r, &req); err != nil {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Invalid request body",
		})
		return
	}

	// Validate refresh token is not empty
	if req.RefreshToken == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "Refresh token is required",
		})
		return
	}

	// Invalidate the refresh token
	InvalidateRefreshToken(req.RefreshToken)

	// In production: delete token from database and clear user session

	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
		Data: model.ResponseData{
			ProcessedAt: "",
		},
	})
}
