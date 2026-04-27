package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Taterbro/backendStageZero/internal/model"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// GetGitHubAuthURL generates the GitHub OAuth authorization URL
func GetGitHubAuthURL(challenge string) string {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	redirectURI := os.Getenv("GITHUB_REDIRECT_URI")
	scope := "user:email"

	// GitHub doesn't use PKCE, but we generate state for CSRF protection
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		clientID,
		url.QueryEscape(redirectURI),
		scope,
		challenge,
	)

	return authURL
}

// ExchangeCodeForToken exchanges GitHub authorization code for access token
func ExchangeCodeForToken(code string) (string, error) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return "", err
	}

	req.URL.RawQuery = data.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Failed to unmarshal response: %v", err)
		return "", err
	}

	if _, ok := result["error"]; ok {
		errMsg := result["error"].(string)
		log.Printf("GitHub OAuth error: %s", errMsg)
		return "", fmt.Errorf("github oauth error: %s", errMsg)
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		log.Printf("No access token in response")
		return "", fmt.Errorf("no access token in response")
	}

	return accessToken, nil
}

// GetGitHubUser fetches GitHub user information using access token
func GetGitHubUser(accessToken string) (*model.GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("GitHub API returned status: %d", resp.StatusCode)
		return nil, fmt.Errorf("github api returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return nil, err
	}

	var user model.GitHubUser
	err = json.Unmarshal(body, &user)
	if err != nil {
		log.Printf("Failed to unmarshal user: %v", err)
		return nil, err
	}

	return &user, nil
}
