package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"time"

	"banking-service/internal/analytics/delivery/grpc"
	"banking-service/internal/analytics/repository"
	"banking-service/internal/analytics/usecase"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	analyticsv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/analytics/v1"
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
		dbUrl = "postgres://user:password@localhost:5432/banking_accounts?sslmode=disable"
	}
	accountServiceUrl := os.Getenv("ACCOUNT_SERVICE_URL")
	if accountServiceUrl == "" {
		accountServiceUrl = "localhost:50051"
	}

	// 1. Run migrations for analytics
	migrationUrl := dbUrl
	if strings.Contains(migrationUrl, "?") {
		migrationUrl += "&x-migrations-table=analytics_schema_migrations"
	} else {
		migrationUrl += "?x-migrations-table=analytics_schema_migrations"
	}
	m, err := migrate.New("file://migrations/analytics", migrationUrl)
	if err != nil {
		log.Fatalf("Unable to create migration instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Analytics database migrations applied successfully")

	// 2. Initialize pgx pool
	poolConfig, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		log.Fatalf("Unable to parse database config: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// 3. Connect to Account Service gRPC Client
	log.Printf("Connecting to Account Service at %s...", accountServiceUrl)
	accountConn, err := googlegrpc.NewClient(accountServiceUrl, googlegrpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Account Service: %v", err)
	}
	defer accountConn.Close()
	accountClient := accountv1.NewAccountServiceClient(accountConn)

	// 4. Wire repository and usecase
	repo := repository.NewAnalyticsRepository(pool)
	uc := usecase.NewAnalyticsUseCase(repo, accountClient)
	handler := grpc.NewAnalyticsHandler(uc)

	// 5. Start gRPC Server
	listener, err := net.Listen("tcp", ":50054")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := googlegrpc.NewServer()
	analyticsv1.RegisterAnalyticsServiceServer(grpcServer, handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Println("Analytics Service running on :50054")
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
		log.Println("Analytics Service stopped cleanly.")
	}
}
