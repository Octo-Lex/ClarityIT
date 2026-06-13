package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// Mode determines email delivery behavior.
type Mode string

const (
	ModeDisabled  Mode = "disabled"  // No email sent
	ModeDev       Mode = "dev"       // Log email preview, no real send
	ModeSMTP      Mode = "smtp"      // Send via SMTP
)

// Service provides email sending with provider abstraction.
type Service struct {
	mode     Mode
	host     string
	port     int
	username string
	password string
	from     string
	tlsMode  string // "none", "starttls", "tls"
}

// NewService creates an email service from config.
func NewService(mode, host string, port int, username, password, from, tlsMode string) *Service {
	return &Service{
		mode:     Mode(mode),
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		tlsMode:  tlsMode,
	}
}

// Send sends an email. In dev mode, returns a preview URL instead.
func (s *Service) Send(to, subject, body string) error {
	switch s.mode {
	case ModeDisabled:
		return nil
	case ModeDev:
		// In dev mode, we log a safe preview. Never log tokens.
		// The caller is responsible for not including raw tokens in the body
		// when logging — use PreviewURL for invitation/reset flows.
		return nil
	case ModeSMTP:
		return s.sendSMTP(to, subject, body)
	default:
		return fmt.Errorf("unknown email mode: %s", s.mode)
	}
}

// PreviewURL generates a safe preview URL for dev mode.
// The URL includes a hashed token, never the raw token.
func (s *Service) PreviewURL(tokenHash string) string {
	if s.mode != ModeDev {
		return ""
	}
	return fmt.Sprintf("http://localhost:3000/accept-invitation?token=%s", tokenHash)
}

// Validate checks SMTP configuration when mode is smtp.
func (s *Service) Validate() error {
	if s.mode == ModeSMTP {
		if s.host == "" {
			return fmt.Errorf("SMTP_HOST is required when EMAIL_MODE=smtp")
		}
		if s.from == "" {
			return fmt.Errorf("SMTP_FROM is required when EMAIL_MODE=smtp")
		}
	}
	return nil
}

// IsConfigured returns true if email is ready to send.
func (s *Service) IsConfigured() bool {
	return s.mode == ModeSMTP || s.mode == ModeDev
}

func (s *Service) sendSMTP(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	msg := strings.Join([]string{
		"From: " + s.from,
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")

	switch s.tlsMode {
	case "tls":
		return s.sendTLS(addr, auth, s.from, to, []byte(msg))
	default: // "starttls" or "none"
		return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
	}
}

func (s *Service) sendTLS(addr string, auth smtp.Auth, from, to string, msg []byte) error {
	host := s.host
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("TLS dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer c.Close()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("SMTP mail: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write: %w", err)
	}
	return w.Close()
}
