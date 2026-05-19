package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"banking-service/internal/transaction/delivery/grpc"
	"banking-service/internal/transaction/repository"
	"banking-service/internal/transaction/usecase"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	transactionv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/transaction/v1"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://user:password@localhost:5433/banking_transactions?sslmode=disable"
	}
	accountServiceUrl := os.Getenv("ACCOUNT_SERVICE_URL")
	if accountServiceUrl == "" {
		accountServiceUrl = "localhost:50051"
	}

	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}

	// Run auto migrations
	m, err := migrate.New("file://migrations/transaction", dbUrl)
	if err != nil {
		log.Fatalf("Unable to create migration instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations applied successfully")

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Connect to Account Service (gRPC Client)
	log.Printf("Connecting to Account Service at %s...", accountServiceUrl)
	accountConn, err := googlegrpc.NewClient(accountServiceUrl, googlegrpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Account Service: %v", err)
	}
	defer accountConn.Close()
	accountClient := accountv1.NewAccountServiceClient(accountConn)

	// Repositories
	txManager := repository.NewTransactionManager(pool)
	repo := repository.NewTransactionRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	idemRepo := repository.NewIdempotencyRepository(pool)

	// Usecase
	uc := usecase.NewTransactionUseCase(txManager, repo, outboxRepo, idemRepo, accountClient)

	// Delivery
	handler := grpc.NewTransactionHandler(uc)

	listener, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := googlegrpc.NewServer()
	transactionv1.RegisterTransactionServiceServer(grpcServer, handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Println("Starting Transaction Service gRPC server on :50052...")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-time.After(10 * time.Second):
		log.Println("Timeout reached, forcing stop...")
		grpcServer.Stop()
	case <-stopped:
		log.Println("Server stopped cleanly.")
	}
}
