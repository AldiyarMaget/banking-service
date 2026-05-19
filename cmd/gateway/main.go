package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"banking-service/internal/gateway"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	transactionv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/transaction/v1"
	userv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/user/v1"
	analyticsv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/analytics/v1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	accountUrl := os.Getenv("ACCOUNT_SERVICE_URL")
	if accountUrl == "" {
		accountUrl = "localhost:50051"
	}
	txUrl := os.Getenv("TRANSACTION_SERVICE_URL")
	if txUrl == "" {
		txUrl = "localhost:50052"
	}
	userUrl := os.Getenv("USER_SERVICE_URL")
	if userUrl == "" {
		userUrl = "localhost:50053"
	}
	analyticsUrl := os.Getenv("ANALYTICS_SERVICE_URL")
	if analyticsUrl == "" {
		analyticsUrl = "localhost:50054"
	}

	// Connect to Account Service
	log.Printf("Connecting to Account Service at %s...", accountUrl)
	accConn, err := grpc.NewClient(accountUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Account Service: %v", err)
	}
	defer accConn.Close()
	accountClient := accountv1.NewAccountServiceClient(accConn)

	// Connect to Transaction Service
	log.Printf("Connecting to Transaction Service at %s...", txUrl)
	txConn, err := grpc.NewClient(txUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Transaction Service: %v", err)
	}
	defer txConn.Close()
	txClient := transactionv1.NewTransactionServiceClient(txConn)

	// Connect to User Service
	log.Printf("Connecting to User Service at %s...", userUrl)
	userConn, err := grpc.NewClient(userUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to User Service: %v", err)
	}
	defer userConn.Close()
	userClient := userv1.NewUserServiceClient(userConn)

	// Connect to Analytics Service
	log.Printf("Connecting to Analytics Service at %s...", analyticsUrl)
	analyticsConn, err := grpc.NewClient(analyticsUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Analytics Service: %v", err)
	}
	defer analyticsConn.Close()
	analyticsClient := analyticsv1.NewAnalyticsServiceClient(analyticsConn)

	// Setup Router
	r := chi.NewRouter()

	// CORS Middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"}, // Allows all
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Correlation-ID"},
		ExposedHeaders:   []string{"Link", "X-Correlation-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Tracing and Logging Middlewares
	r.Use(gateway.CorrelationIDMiddleware)
	r.Use(gateway.LoggerMiddleware)

	// Handlers
	handler := gateway.NewHandler(accountClient, txClient, userClient, analyticsClient)
	handler.RegisterRoutes(r)

	// Server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Starting API Gateway on :8080...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway server error: %v", err)
		}
	}()

	// Graceful Shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("Received shutdown signal. Shutting down Gateway...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Gateway Shutdown Failed: %+v", err)
	}
	log.Println("Gateway stopped cleanly.")
}
