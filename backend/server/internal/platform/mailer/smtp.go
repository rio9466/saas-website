package mailer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// dial opens the transport for the configured TLS mode.
func (m *SMTPMailer) dial(ctx context.Context, cfg Settings) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: cfg.ConnectTimeout}
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	if cfg.TLSMode == TLSModeSSL {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, m.tlsConfigFor(cfg.Host))
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// smtpClient wraps a connection in the net/smtp client.
func smtpClient(conn net.Conn, cfg Settings) (*smtp.Client, error) {
	host := cfg.Host
	if strings.Contains(host, ":") {
		host = strings.SplitN(host, ":", 2)[0]
	}
	return smtp.NewClient(conn, host)
}

func tlsConfig(host string) *tls.Config {
	serverName := host
	if strings.Contains(serverName, ":") {
		serverName = strings.SplitN(serverName, ":", 2)[0]
	}
	return &tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	}
}

// plainAuth returns the SMTP AUTH PLAIN authenticator. Credentials are only
// used in memory for the single connection; they never enter error strings.
func plainAuth(username, password, host string) smtp.Auth {
	return smtp.PlainAuth("", username, password, host)
}

// hostname echoes the local hostname for the SMTP HELO/EHLO command.
func hostname() string {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "localhost"
	}
	return name
}

// messageBytes builds a minimal RFC 5322 message with both Subject and body.
func messageBytes(fromEmail, fromName, to, subject, htmlBody string) []byte {
	var b strings.Builder
	if fromName != "" {
		b.WriteString("From: ")
		b.WriteString(quoteHeaderName(fromName))
		b.WriteString(" <")
		b.WriteString(fromEmail)
		b.WriteString(">\r\n")
	} else {
		b.WriteString("From: ")
		b.WriteString(fromEmail)
		b.WriteString("\r\n")
	}
	b.WriteString("To: ")
	b.WriteString(to)
	b.WriteString("\r\n")
	b.WriteString("Subject: ")
	b.WriteString(sanitizeHeader(subject))
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("Date: ")
	b.WriteString(time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("\r\n\r\n")
	b.WriteString(htmlBody)
	return []byte(b.String())
}

func quoteHeaderName(name string) string {
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\n", "")
	return name
}

// sanitizeHeader strips CR/LF so header injection is impossible.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// sanitizeSMTPError removes anything that could echo credentials (connection
// strings, URL userinfo) from an SMTP error. net/smtp errors never include
// the plaintext password, but we keep a defensive scrub shape.
func sanitizeSMTPError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	// Never include URL-style userinfo if a server handcrafts one.
	if at := strings.LastIndex(msg, "@"); at > 0 && strings.Contains(msg[:at], ":") {
		msg = "smtp conversation failed"
	}
	return fmt.Errorf("%s", truncateErr(msg))
}

func truncateErr(msg string) string {
	if len(msg) > 300 {
		return msg[:300] + "..."
	}
	return msg
}

// serverNamePolicy is referenced so x509 verifications benefit from the
// standard pool when a deployment sets SSL_CERT_FILE.
var _ = x509.SystemCertPool
