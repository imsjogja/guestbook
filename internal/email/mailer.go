// Package email provides the SMTP transport used by authentication and
// communication workflows.
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"guestflow/internal/config"
)

// Mailer sends a plain-text email.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPMailer sends mail using STARTTLS on port 587 or implicit TLS on port 465.
type SMTPMailer struct {
	cfg config.EmailConfig
}

func NewSMTPMailer(cfg config.EmailConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	if !m.cfg.Enabled {
		return fmt.Errorf("email delivery is disabled")
	}
	if strings.TrimSpace(m.cfg.Host) == "" || strings.TrimSpace(m.cfg.User) == "" || strings.TrimSpace(m.cfg.Password) == "" || strings.TrimSpace(m.cfg.From) == "" {
		return fmt.Errorf("SMTP configuration is incomplete")
	}

	port := m.cfg.Port
	if port == 0 {
		port = 587
	}
	address := fmt.Sprintf("%s:%d", m.cfg.Host, port)
	dialer := &net.Dialer{Timeout: 20 * time.Second}
	var conn net.Conn
	var err error
	if port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return fmt.Errorf("connect SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	if port != 465 && m.cfg.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}

	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)
	if ok, _ := client.Extension("AUTH"); !ok {
		return fmt.Errorf("SMTP server does not support AUTH")
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authenticate SMTP client: %w", err)
	}
	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}

	fromName := mime.QEncoding.Encode("UTF-8", m.cfg.FromName)
	from := m.cfg.From
	if strings.TrimSpace(m.cfg.FromName) != "" {
		from = fmt.Sprintf("%s <%s>", fromName, m.cfg.From)
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", from, to, mime.QEncoding.Encode("UTF-8", subject), time.Now().UTC().Format(time.RFC1123Z), body)
	if _, err := writer.Write([]byte(message)); err != nil {
		writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("send SMTP message: %w", err)
	}
	return client.Quit()
}
