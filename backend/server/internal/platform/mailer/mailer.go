// Package mailer sends transactional email through the configured SMTP
// endpoint. TLS mode, credentials, and timeouts are validated up front so a
// misconfigured mailer fails at construction, not silently on first send.
// Send errors never echo the SMTP password or ciphertext.
package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Connection TTL constants for SMTP dialing and conversation.
const (
	defaultConnectTimeout = 10 * time.Second
	defaultWriteTimeout   = 10 * time.Second
)

// Supported TLS modes mirror the domain system-settings values.
const (
	TLSModeNone     = "none"
	TLSModeStartTLS = "starttls"
	TLSModeSSL      = "ssl"
)

var (
	// ErrInvalidSettings reports a mailer configuration that can never work.
	ErrInvalidSettings = errors.New("invalid smtp settings")
	// ErrSendFailed reports a failed SMTP conversation with a sanitized cause.
	ErrSendFailed = errors.New("send smtp mail failed")
)

// Settings describes one SMTP endpoint. Password is the decrypted plaintext
// SMTP password, provided only at send/build time from the secret box.
type Settings struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	TLSMode   string
	// ConnectTimeout bounds dialing; zero uses the default.
	ConnectTimeout time.Duration
	// WriteTimeout bounds the SMTP conversation; zero uses the default.
	WriteTimeout time.Duration
}

// ValidateSettings checks endpoint invariants without connecting.
func ValidateSettings(cfg Settings) error {
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("%w: smtp host is required", ErrInvalidSettings)
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("%w: smtp port must be 1-65535", ErrInvalidSettings)
	}
	if strings.TrimSpace(cfg.FromEmail) == "" {
		return fmt.Errorf("%w: smtp from email is required", ErrInvalidSettings)
	}
	switch cfg.TLSMode {
	case TLSModeNone:
		if strings.TrimSpace(cfg.Username) != "" {
			return fmt.Errorf("%w: smtp username requires starttls or ssl encryption", ErrInvalidSettings)
		}
	case TLSModeStartTLS, TLSModeSSL:
		// Authenticated and anonymous encrypted delivery both allowed.
	default:
		return fmt.Errorf("%w: smtp tls mode must be none, starttls, or ssl", ErrInvalidSettings)
	}
	return nil
}

// Mailer sends one email message.
type Mailer interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// SMTPMailer is a Mailer backed by the net/smtp client.
type SMTPMailer struct {
	cfg Settings
	// tlsOverride is test-only: it replaces TLS verification for in-process
	// fake SMTP servers whose certificates are not in the system roots.
	tlsOverride *tls.Config
}

// New validates and constructs an SMTPMailer.
func New(cfg Settings) (*SMTPMailer, error) {
	cfg = normalize(cfg)
	if err := ValidateSettings(cfg); err != nil {
		return nil, err
	}
	return &SMTPMailer{cfg: cfg}, nil
}

// MustNew panics unless the settings are valid; intended for tests and
// startup wiring where validation already failed hard.
func MustNew(cfg Settings) *SMTPMailer {
	m, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return m
}

// setTLSOverrideForTest replaces TLS verification; tests use it with the
// fake in-process SMTP server. Production code never sets this.
func (m *SMTPMailer) setTLSOverrideForTest(cfg *tls.Config) {
	m.tlsOverride = cfg
}

// tlsConfigFor returns the TLS config for the SMTP host, honoring the
// optional test override.
func (m *SMTPMailer) tlsConfigFor(host string) *tls.Config {
	if m != nil && m.tlsOverride != nil {
		clone := m.tlsOverride.Clone()
		if clone.ServerName == "" {
			clone.ServerName = host
		}
		return clone
	}
	return tlsConfig(host)
}

func normalize(cfg Settings) Settings {
	if strings.TrimSpace(cfg.TLSMode) == "" {
		cfg.TLSMode = TLSModeStartTLS
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.FromEmail = strings.TrimSpace(cfg.FromEmail)
	cfg.FromName = strings.TrimSpace(cfg.FromName)
	return cfg
}

// Send delivers one HTML email. The context deadline bounds the whole
// conversation; connection errors are sanitized and never include credentials.
func (m *SMTPMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	if m == nil {
		return ErrSendFailed
	}
	cfg := normalize(m.cfg)

	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("%w: recipient is required", ErrSendFailed)
	}

	deadline := time.Now().Add(cfg.WriteTimeout)
	conn, err := m.dial(ctx, cfg)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, sanitizeSMTPError(err))
	}
	defer conn.Close()
	_ = conn.SetDeadline(deadline)

	client, err := smtpClient(conn, cfg)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, sanitizeSMTPError(err))
	}
	defer client.Close()

	if err := client.Hello(hostname()); err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, sanitizeSMTPError(err))
	}

	if cfg.TLSMode == TLSModeStartTLS {
		if err := client.StartTLS(m.tlsConfigFor(cfg.Host)); err != nil {
			return fmt.Errorf("%w: starttls: %v", ErrSendFailed, sanitizeSMTPError(err))
		}
	}
	if strings.TrimSpace(cfg.Username) != "" {
		auth := plainAuth(cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("%w: smtp auth: %v", ErrSendFailed, sanitizeSMTPError(err))
		}
	}

	if err := client.Mail(cfg.FromEmail); err != nil {
		return fmt.Errorf("%w: mail from: %v", ErrSendFailed, sanitizeSMTPError(err))
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("%w: rcpt to: %v", ErrSendFailed, sanitizeSMTPError(err))
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("%w: data: %v", ErrSendFailed, sanitizeSMTPError(err))
	}
	if _, err := w.Write(messageBytes(cfg.FromEmail, cfg.FromName, to, subject, htmlBody)); err != nil {
		_ = w.Close()
		return fmt.Errorf("%w: write message: %v", ErrSendFailed, sanitizeSMTPError(err))
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("%w: finish message: %v", ErrSendFailed, sanitizeSMTPError(err))
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("%w: quit: %v", ErrSendFailed, sanitizeSMTPError(err))
	}
	return nil
}
