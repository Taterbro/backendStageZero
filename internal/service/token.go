package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// PKCEChallenge represents a PKCE challenge
type PKCEChallenge struct {
	Verifier  string
	Challenge string
}

// TokenPair holds both access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// GeneratePKCEChallenge creates a PKCE challenge
func GeneratePKCEChallenge() (*PKCEChallenge, error) {
	// Generate random verifier (43-128 characters)
	verifier := make([]byte, 32)
	_, err := rand.Read(verifier)
	if err != nil {
		return nil, err
	}

	// Base64 URL encode without padding
	encodedVerifier := base64.RawURLEncoding.EncodeToString(verifier)

	// Create challenge from verifier using SHA256
	hash := sha256.Sum256([]byte(encodedVerifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEChallenge{
		Verifier:  encodedVerifier,
		Challenge: challenge,
	}, nil
}

// GenerateStateToken creates a secure random state token
func GenerateStateToken() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes), nil
}

// GenerateTokenPair creates JWT-like tokens with expiration
// For production, use a proper JWT library like github.com/golang-jwt/jwt
func GenerateTokenPair(userID string) (*TokenPair, error) {
	accessToken := fmt.Sprintf("access_%s_%d", userID, time.Now().UnixNano())
	refreshToken := fmt.Sprintf("refresh_%s_%d", userID, time.Now().UnixNano())

	accessExpiresAt := time.Now().Add(3 * time.Minute)

	// For production, implement proper JWT signing
	// This is a simplified version - use github.com/golang-jwt/jwt for real implementation

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
	}, nil
}

// ValidateRefreshToken checks if refresh token is still valid
// This should be backed by a database in production
func ValidateRefreshToken(token string) (string, error) {
	// Parse token and extract user ID
	// In production, validate JWT signature and expiration
	if token == "" {
		return "", fmt.Errorf("invalid refresh token")
	}
	return token, nil
}
