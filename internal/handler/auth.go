package handler

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Taterbro/backendStageZero/internal/database"
	"github.com/Taterbro/backendStageZero/internal/model"
	"github.com/Taterbro/backendStageZero/internal/utils"
	"github.com/google/uuid"
)

type CodeVerifier struct {
	Value     string
	ExpiresAt time.Time
}

var cache = map[string]CodeVerifier{}

func Set(key, value string, ttl time.Duration) {
	cache[key] = CodeVerifier{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func Get(key string) (string, bool) {
	item, ok := cache[key]
	if !ok {
		return "", false
	}

	if time.Now().After(item.ExpiresAt) {
		delete(cache, key)
		return "", false
	}

	return item.Value, true
}

func GitHubAuth(w http.ResponseWriter, r *http.Request) {
	client_id := os.Getenv("GITHUB_CLIENT_ID")
	redirect_uri := os.Getenv("GITHUB_REDIRECT_URI")
	code_verifier, err := utils.GenerateToken(32)
	if err != nil {
		log.Println("couldn't generate code challenge")
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Something went wrong on our end",
		})
		return
	}
	hash := sha256.Sum256([]byte(code_verifier))
	code_challenge := base64.RawURLEncoding.EncodeToString(hash[:])
	code_challenge_method := "S256"
	state, err := utils.GenerateToken(16)
	if err != nil {
		log.Println("couldn't generate state")
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "Something went wrong on our end",
		})
		return
	}
	Set(state, code_verifier, 10*time.Minute)

	fullUrl := fmt.Sprintf("%s/authorize?client_id=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=%s&state=%s", os.Getenv("GITHUB_URL"), client_id, redirect_uri, code_challenge, code_challenge_method, state)
	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
		Data:   fullUrl,
	})

}

func GitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if state == "" {
		utils.WriteJson(w, http.StatusBadRequest, model.ErrorResponse{
			Status:  "error",
			Message: "missing state parameter",
		})
		return
	}
	codeVerifier, ok := Get(state)
	if !ok {
		utils.WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: "Invalid state; unauthorized",
		})
		return
	}
	redirect_uri := "http://localhost:8080/auth/github/callback" //temporary for now
	fullUrl := fmt.Sprintf("%s/access_token", os.Getenv("GITHUB_URL"))
	payload := url.Values{}
	payload.Set("client_id", os.Getenv("GITHUB_CLIENT_ID"))
	payload.Set("client_secret", os.Getenv("GITHUB_CLIENT_SECRET"))
	payload.Set("redirect_uri", redirect_uri)
	payload.Set("code", code)
	payload.Set("code_verifier", codeVerifier)
	req, _ := http.NewRequest("POST", fullUrl, strings.NewReader(payload.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("error posting full url: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong while trying to fetch github access tokens",
		})
		return
	}
	defer resp.Body.Close()
	var data model.GitAuthResponse
	decoder := json.NewDecoder(resp.Body)

	if err = decoder.Decode(&data); err != nil {
		log.Println("error decoding response body: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong while trying to fetch github access tokens",
		})
		return
	}
	if data.Error != "" {
		log.Println("error: ", data.Error, "\nerror_description: ", data.ErrorDescription)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong while trying to fetch github access tokens",
		})
		return
	}
	log.Println("access token: ", data.AccessToken)
	var userResp *http.Request
	userResp, err = http.NewRequest("GET", fmt.Sprintf("%s/user", os.Getenv("GITHUB_ROOT")), nil)
	if err != nil {
		log.Println(err)
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong while trying to fetch github user profile",
		})
		return
	}
	userResp.Header.Set("Authorization", fmt.Sprintf("Bearer %s", data.AccessToken))
	client := &http.Client{}
	user, err := client.Do(userResp)
	if err != nil {
		log.Println("err")
		utils.WriteJson(w, http.StatusBadGateway, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong while trying to fetch github user profile",
		})
		return
	}
	defer user.Body.Close()
	var userDetails model.GitUserResponse
	userDecoder := json.NewDecoder(user.Body)
	err = userDecoder.Decode(&userDetails)
	if err != nil {
		log.Println("error while decoding user details: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong while trying to fetch github access tokens",
		})
		return
	}
	userExists, err := database.GetAccount(database.GetAccountType{GithubId: userDetails.Id})
	var activeId string
	if err != nil {
		log.Println("find account error: ", err)
		var userObject = database.Account{
			ID:          uuid.New().String(),
			GitHubID:    userDetails.Id,
			Username:    userDetails.Name,
			Email:       userDetails.Email,
			AvatarURL:   userDetails.AvatarUrl,
			Role:        "analyst",
			IsActive:    true,
			LastLoginAt: time.Now(),
			CreatedAt:   time.Now(),
		}
		accountId, err := database.AddAccount(userObject)
		activeId = accountId
		if err != nil {
			log.Println("error creating user acccount: ", err)
			utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
				Status:  "error",
				Message: "something went wrong on our end",
			})
			return
		}
		database.UpdateLoginTime(database.GetAccountType{Id: activeId})
	} else {
		activeId = userExists.ID
		database.UpdateLoginTime(database.GetAccountType{Id: activeId})
	}

	refreshToken, err := utils.GenerateToken(32)
	if err != nil {
		log.Println("error generating token: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong on our end",
		})
		return
	}
	accessToken, err := utils.GenerateAccessToken(activeId)
	if err != nil {
		log.Println("error generating access token: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong on our end",
		})
		return
	}
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(refreshToken)))
	database.AddRefreshToken(tokenHash, activeId)

	// http.SetCookie(w, &http.Cookie{
	// 	Name:     "access_token",
	// 	Value:    accessToken,
	// 	Path:     "/",
	// 	HttpOnly: true,
	// 	Secure:   true,
	// 	SameSite: http.SameSiteLaxMode,
	// 	MaxAge:   180,
	// })

	// http.SetCookie(w, &http.Cookie{
	// 	Name:     "refresh_token",
	// 	Value:    tokenHash,
	// 	Path:     "/",
	// 	HttpOnly: true,
	// 	Secure:   true,
	// 	SameSite: http.SameSiteLaxMode,
	// 	MaxAge:   300,
	// })

	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
		Data: map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		log.Println("decoding error: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong on our end",
		})
		return
	}
	token := body.RefreshToken
	if token == "" {
		utils.WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: "no token provided",
		})
		return
	}
	hashedBytes := sha256.Sum256([]byte(token))
	hash := fmt.Sprintf("%x", hashedBytes)
	tokenVerify, err := database.GetRefreshToken(database.TokenGetter{Hash: hash})
	if err != nil {
		log.Println("get token error: ", err)
		utils.WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: "invalid token",
		})
		return
	}

	if !time.Now().Before(tokenVerify.ExpiresAt) {
		utils.WriteJson(w, http.StatusUnauthorized, model.ErrorResponse{
			Status:  "error",
			Message: "token expired",
		})
		return
	}
	err = database.DeleteRefreshToken(database.TokenGetter{Hash: tokenVerify.TokenHash})
	if err != nil {
		log.Println("delete token error: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong on our end",
		})
		return
	}

	refreshToken, err := utils.GenerateToken(32)
	if err != nil {
		log.Println("error generating token: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong on our end",
		})
		return
	}
	accessToken, err := utils.GenerateAccessToken(tokenVerify.UserID)
	if err != nil {
		log.Println("error generating access token: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "something went wrong on our end",
		})
		return
	}
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(refreshToken)))
	database.AddRefreshToken(tokenHash, tokenVerify.UserID)

	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
		Data: map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})

}

func Logout(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	parts := strings.Split(header, " ")
	token := parts[1]
	userId, err := utils.GetUserIDFromToken(token)
	if err != nil {
		log.Println("find user id error: ", err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "we could not find the account associated with that token",
		})
		return
	}
	err = database.DeleteRefreshToken(database.TokenGetter{UserId: userId})
	if err != nil {
		log.Println(err)
		utils.WriteJson(w, http.StatusInternalServerError, model.ErrorResponse{
			Status:  "error",
			Message: "token invalidation failed; logout was unsuccessful",
		})
		return
	}
	utils.WriteJson(w, http.StatusOK, model.SuccessResponse{
		Status: "success",
	})
}
