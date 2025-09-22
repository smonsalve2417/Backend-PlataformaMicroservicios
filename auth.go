package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"

func WithJWTAuth(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := GetTokenFromRequest(r)

		jwtResp, err := ValidateJWT(tokenString)
		if err != nil {
			log.Printf("failed to validate token %v", err)
			PermissionDenied(w)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, jwtResp.User.Sub)

		handlerFunc(w, r.WithContext(ctx))

	}
}

func PermissionDenied(w http.ResponseWriter) {
	WriteError(w, http.StatusForbidden, "permission denied")
}

func GetTokenFromRequest(r *http.Request) string {
	tokenAuth := r.Header.Get("Authorization")
	tokenQuery := r.URL.Query().Get("token")

	if tokenAuth != "" {
		if len(tokenAuth) > 7 && tokenAuth[:7] == "Bearer " {
			return tokenAuth[7:]
		}
		return tokenAuth
	}

	if tokenQuery != "" {
		return tokenQuery
	}

	return ""
}

func ValidateJWT(tokenString string) (*verifyTokenResponse, error) {
	// Create request
	req, err := http.NewRequest("GET", url+"/auth/"+proyectID+"/verify-token", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	// Add header
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status:", resp.Status)
	fmt.Println("Response:", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	var jwtResp verifyTokenResponse
	if err := json.Unmarshal(body, &jwtResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &jwtResp, nil
}

func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return userID, fmt.Errorf("user ID not found in context or not a string: " + string(userIDKey))
	}
	return userID, nil
}
