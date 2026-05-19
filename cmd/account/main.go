package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"banking-service/internal/account/delivery/grpc"
	"banking-service/internal/account/infrastructure/nats"
	"banking-service/internal/account/repository"
	"banking-service/internal/account/usecase"
	"banking-service/internal/account/worker"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	googlegrpc "google.golang.org/grpc"
)

func main() {
	// Read configuration from environment
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://user:password@localhost:5432/banking_accounts?sslmode=disable"
	}
	natsUrl := os.Getenv("NATS_URL")
	if natsUrl == "" {
		natsUrl = "nats://localhost:4222"
	}

	// Init pgx pool
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Init NATS JetStream
	jsClient, err := nats.InitJetStream(natsUrl)
	if err != nil {
		log.Fatalf("Unable to connect to NATS: %v", err)
	}
	defer jsClient.Close()

	// Initialize Infrastructure Layer (Repositories)
	txManager := repository.NewTransactionManager(pool)
	accountRepo := repository.NewAccountRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	idemRepo := repository.NewIdempotencyRepository(pool)

	// Initialize Application Layer (UseCase)
	accountUC := usecase.NewAccountUseCase(txManager, accountRepo, outboxRepo, idemRepo)

	// Initialize Delivery Layer (gRPC)
	handler := grpc.NewAccountHandler(accountUC)

	// Initialize Reliability Layer (Worker)
	outboxRelay := worker.NewOutboxRelay(txManager, outboxRepo, jsClient)

	// Setup network listener
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create gRPC Server and register handler
	grpcServer := googlegrpc.NewServer()
	accountv1.RegisterAccountServiceServer(grpcServer, handler)

	// Graceful Shutdown Setup
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Start Worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		outboxRelay.Start(ctx)
	}()

	// Start gRPC Server
	go func() {
		log.Println("Starting gRPC server on :50051...")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for OS signal
	<-ctx.Done()
	log.Println("Received shutdown signal. Shutting down gracefully...")

	// Close the gRPC server allowing active connections to finish
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		wg.Wait() // Wait for worker to finish current iteration
		close(stopped)
	}()

	// Safety timeout in case requests or worker hang
	select {
	case <-time.After(10 * time.Second):
		log.Println("Timeout reached, forcing stop...")
		grpcServer.Stop()
	case <-stopped:
		log.Println("gRPC Server and Worker stopped cleanly.")
	}
	
	log.Println("Exiting application...")
}
