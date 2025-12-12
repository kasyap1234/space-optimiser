package main

import (
	"net/http"
	"os"
)

// RapidAPIMiddleware verifies that requests are coming from RapidAPI
// by checking the X-RapidAPI-Proxy-Secret header against the configured secret.
func RapidAPIMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the expected secret from environment variable
		expectedSecret := os.Getenv("RAPIDAPI_PROXY_SECRET")

		// If no secret is configured, skip validation (useful for local development)
		if expectedSecret == "" {
			next(w, r)
			return
		}

		// Get the secret from the request header
		proxySecret := r.Header.Get("X-RapidAPI-Proxy-Secret")

		// Verify the secret matches
		if proxySecret != expectedSecret {
			http.Error(w, "Unauthorized: Invalid or missing RapidAPI proxy secret", http.StatusUnauthorized)
			return
		}

		// Request is valid, proceed to the next handler
		next(w, r)
	}
}

