package main

//Don't touch this file

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/go-playground/validator"
)

var Validate = validator.New()

// formatValidationErrors formats a slice of validation errors into a single string
func FormatValidationErrors(validationErrors validator.ValidationErrors) string {
	if len(validationErrors) == 0 {
		return ""
	}
	// Convert the slice of validation errors to a single error message
	errorMessages := make([]string, len(validationErrors))
	for i, err := range validationErrors {
		errorMessages[i] = fmt.Sprintf("Field '%s' failed validation with error: %s", err.Field(), err.Tag())
	}
	return strings.Join(errorMessages, ", ")
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)

}

func ParseJSON(r *http.Request, payload any) error {
	if r.Body == nil {
		return fmt.Errorf("missing request body")
	}
	return json.NewDecoder(r.Body).Decode(payload)
}

func WriteError(w http.ResponseWriter, status int, err string) {
	WriteJSON(w, status, map[string]string{"error": err})
}

func WaitForDocker(cli *client.Client, timeout time.Duration) error {
	ctx := context.Background()
	deadline := time.Now().Add(timeout)

	for {
		_, err := cli.Ping(ctx)
		if err == nil {
			return nil // Docker is alive
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("docker did not become ready within %v", timeout)
		}

		fmt.Println("⏳ Waiting for Docker to start...")
		time.Sleep(1 * time.Second)
	}
}
