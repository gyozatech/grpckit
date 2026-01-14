package main

import (
	"io"
	"log"
	"net/http"
)

// ============================================================
// Webhook Example: Custom HTTP endpoint with dedicated middleware
// This endpoint is NOT in proto - pure HTTP, no gRPC exposure
// ============================================================

// webhookAuthMiddleware validates webhook signatures.
// This middleware ONLY applies to the webhook endpoint, not other endpoints.
func webhookAuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate webhook signature (e.g., from GitHub, Stripe, etc.)
			signature := r.Header.Get("X-Webhook-Signature")
			if signature == "" {
				http.Error(w, "Missing webhook signature", http.StatusUnauthorized)
				return
			}

			// In real code: validate HMAC signature with secret
			// For demo, just check against a known value
			if signature != secret {
				http.Error(w, "Invalid webhook signature", http.StatusForbidden)
				return
			}

			log.Printf("[Webhook Auth] Signature validated for %s", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}

// webhookHandler handles incoming webhooks.
// It can accept any payload format - not constrained by proto.
func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the webhook payload (could be JSON, form data, XML, etc.)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Process webhook (e.g., GitHub push event, Stripe payment, etc.)
	log.Printf("[Webhook] Received payload (%d bytes): %s", len(body), string(body))

	// Respond with any format - not constrained by proto
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "received", "message": "Webhook processed successfully"}`))
}
