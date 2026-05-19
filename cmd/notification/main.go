package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"banking-service/internal/notification/infrastructure/nats"
	"banking-service/internal/notification/infrastructure/smtp"
	"banking-service/internal/notification/usecase"
	natsgo "github.com/nats-io/nats.go"
)

func main() {
	natsUrl := os.Getenv("NATS_URL")
	if natsUrl == "" {
		natsUrl = "nats://localhost:4222"
	}

	// 1. Connect to NATS JetStream
	nc, err := natsgo.Connect(natsUrl)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	// Create BANKING stream if not exists (allows notification service to boot first)
	_, err = js.AddStream(&natsgo.StreamConfig{
		Name:     "BANKING",
		Subjects: []string{"banking.*"},
	})
	if err != nil {
		log.Printf("Note: Stream initialization info: %v", err)
	}

	// 2. Initialize SMTP Adapter
	mailer := smtp.NewSmtpAdapter()

	// 3. Initialize UseCase
	uc := usecase.NewNotificationUseCase(mailer)

	// 4. Initialize NATS Consumer
	consumer := nats.NewConsumer(js, uc)

	// Graceful Shutdown context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	log.Println("Starting Notification Service workers...")

	// Start Consumer for Account Creation
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := consumer.Start(ctx, "banking.account.created", "notification-account"); err != nil {
			log.Printf("Consumer account.created stopped with error: %v", err)
		}
	}()

	// Start Consumer for Transactions
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := consumer.Start(ctx, "banking.transaction.completed", "notification-transaction"); err != nil {
			log.Printf("Consumer transaction.completed stopped with error: %v", err)
		}
	}()

	// Wait for OS signal
	<-ctx.Done()
	log.Println("Received shutdown signal. Waiting for workers to finish current messages...")
	
	// Wait for workers to finish their current processing loop and NATS Ack
	wg.Wait()
	log.Println("Notification Service stopped cleanly.")
}
