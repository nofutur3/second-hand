package output

import (
	"fmt"
	"secondHand/src/backend/internal/config"

	"gopkg.in/gomail.v2"
)

// EmailSender handles sending emails
type EmailSender struct {
	cfg *config.SMTPConfig
}

// NewEmailSender creates a new email sender
func NewEmailSender(cfg *config.SMTPConfig) *EmailSender {
	return &EmailSender{cfg: cfg}
}

// SendHTML sends an HTML email
func (s *EmailSender) SendHTML(subject, htmlBody string) error {
	if s.cfg.User == "" || s.cfg.Password == "" {
		return fmt.Errorf("SMTP credentials not configured")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.cfg.From)
	m.SetHeader("To", s.cfg.To)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	var port int
	fmt.Sscanf(s.cfg.Port, "%d", &port)

	d := gomail.NewDialer(s.cfg.Host, port, s.cfg.User, s.cfg.Password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendDiffEmail sends a diff report via email
func (s *EmailSender) SendDiffEmail(keyword, htmlContent string) error {
	subject := fmt.Sprintf("Second Hand: Changes for '%s'", keyword)
	return s.SendHTML(subject, htmlContent)
}

// SendProductsEmail sends a products report via email
func (s *EmailSender) SendProductsEmail(keyword, htmlContent string) error {
	subject := fmt.Sprintf("Second Hand: Results for '%s'", keyword)
	return s.SendHTML(subject, htmlContent)
}
