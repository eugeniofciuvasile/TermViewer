package email

import (
	"fmt"
	"net/smtp"

	"github.com/eugen/termviewer/server/backend/pkg/config"
)

type EmailService struct {
	cfg *config.Config
}

func InitEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		cfg: cfg,
	}
}

func (s *EmailService) SendActivationEmail(to, activationLink string) error {
	if s.cfg.SMTPHost == "" {
		fmt.Printf("[EMAIL LOG] Activation link generated (SMTP not configured)\n")
		return nil
	}

	headerFrom := fmt.Sprintf("From: %s\n", s.cfg.SMTPFrom)
	headerTo := fmt.Sprintf("To: %s\n", to)
	headerSubject := "Subject: Activate your TermViewer Account\n"
	headerMime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	
	body := fmt.Sprintf("<html><body><h2>Welcome to TermViewer</h2><p>Your account has been approved.</p><a href='%s'>Click here to activate your account</a></body></html>", activationLink)
	
	msg := []byte(headerFrom + headerTo + headerSubject + headerMime + body)

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)

	fmt.Printf("[EMAIL] Attempting to send activation email via %s\n", s.cfg.SMTPHost)
	err := smtp.SendMail(addr, auth, s.cfg.SMTPFrom, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	fmt.Printf("[EMAIL SENT] Successfully sent activation email\n")
	return nil
}
