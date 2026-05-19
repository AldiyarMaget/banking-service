package smtp

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"os"

	"banking-service/internal/notification/domain"
)

type SmtpAdapter struct {
	host     string
	port     string
	user     string
	password string
	devMode  bool
}

func NewSmtpAdapter() domain.EmailSender {
	host := os.Getenv("SMTP_HOST")
	devMode := host == "" || host == "dev"

	return &SmtpAdapter{
		host:     host,
		port:     os.Getenv("SMTP_PORT"),
		user:     os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASS"),
		devMode:  devMode,
	}
}

func (a *SmtpAdapter) Send(ctx context.Context, job domain.EmailJob) error {
	if a.devMode {
		log.Printf("[DEV MODE SMTP] CorrelationID: %s | To: %s | Subject: %s\nBody:\n%s\n", 
			job.CorrelationID, job.To, job.Subject, job.HTMLBody)
		return nil
	}

	auth := smtp.PlainAuth("", a.user, a.password, a.host)
	addr := fmt.Sprintf("%s:%s", a.host, a.port)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s", 
		job.To, job.Subject, job.HTMLBody))

	err := smtp.SendMail(addr, auth, a.user, []string{job.To}, msg)
	if err != nil {
		// Transient Error (network issue, rate limit, etc.)
		return fmt.Errorf("transient: failed to send email via SMTP: %w", err)
	}

	return nil
}
