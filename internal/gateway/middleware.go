package gateway

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

// CorrelationIDMiddleware extracts or generates X-Correlation-ID and puts it into gRPC metadata
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := r.Header.Get("X-Correlation-ID")
		if corrID == "" {
			corrID = uuid.NewString()
		}

		// Set header in response for the client
		w.Header().Set("X-Correlation-ID", corrID)

		// Inject into gRPC Outgoing Context
		md := metadata.Pairs("Correlation-ID", corrID)
		ctx := metadata.NewOutgoingContext(r.Context(), md)

		// Inject into HTTP Context for logging
		ctx = context.WithValue(ctx, "Correlation-ID", corrID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggerMiddleware logs basic info about incoming HTTP requests
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID, _ := r.Context().Value("Correlation-ID").(string)
		log.Printf("[Gateway] CorrelationID: %s | %s %s", corrID, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
