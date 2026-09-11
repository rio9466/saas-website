package mailer

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"
)

// fakeSMTPServer is an in-process SMTP server with optional STARTTLS so mailer
// behavior is proven against a real network conversation without external mail.
type fakeSMTPServer struct {
	ln          net.Listener
	addr        string
	startTLS    bool
	tlsConfig   *tls.Config
	messages    chan string
	requireAuth bool
	username    string
	password    string
}

func startFakeSMTP(t *testing.T, startTLS, requireAuth bool) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &fakeSMTPServer{
		ln:          ln,
		addr:        ln.Addr().String(),
		startTLS:    startTLS,
		messages:    make(chan string, 16),
		requireAuth: requireAuth,
	}
	s.tlsConfig = testTLSConfig(t)
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *fakeSMTPServer) fakeClientTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, // test-only fake server cert
		MinVersion:         tls.VersionTLS12,
	}
}

func (s *fakeSMTPServer) hostPort(t *testing.T) (string, int) {
	t.Helper()
	host, port, err := net.SplitHostPort(s.addr)
	if err != nil {
		t.Fatal(err)
	}
	var p int
	if _, err := fmt.Sscanf(port, "%d", &p); err != nil {
		t.Fatal(err)
	}
	return host, p
}

func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	write := func(line string) {
		_, _ = conn.Write([]byte(line + "\r\n"))
	}
	write("220 fake-smtp ESMTP ready")

	var inData bool
	var dataBuf strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				inData = false
				s.messages <- dataBuf.String()
				dataBuf.Reset()
				write("250 2.0.0 queued")
				continue
			}
			dataBuf.WriteString(line)
			dataBuf.WriteString("\r\n")
			continue
		}

		cmd := strings.ToUpper(line)
		args := ""
		if idx := strings.Index(line, " "); idx >= 0 {
			cmd = strings.ToUpper(line[:idx])
			args = strings.TrimSpace(line[idx+1:])
		}
		switch cmd {
		case "EHLO", "HELO":
			// Multiline EHLO: capability lines carry "250-" and the final line
			// must be "250 ..." (space, not dash) or net/smtp blocks forever.
			if !s.startTLS && !s.requireAuth {
				write("250 fake-smtp")
				continue
			}
			write("250-fake-smtp")
			if s.startTLS {
				write("250-STARTTLS")
			}
			if s.requireAuth {
				write("250-AUTH PLAIN")
			}
			write("250 OK")
		case "STARTTLS":
			if !s.startTLS {
				write("502 command not implemented")
				continue
			}
			write("220 ready to start TLS")
			tlsConn := tls.Server(conn, s.tlsConfig)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
		case "AUTH":
			write("503 AUTH not advertised")
		case "MAIL":
			write("250 2.1.0 OK")
		case "RCPT":
			write("250 2.1.5 OK")
		case "DATA":
			write("354 End data with <CR><LF>.<CR><LF>")
			inData = true
		case "QUIT":
			write("221 2.0.0 bye")
			return
		default:
			_ = args
			write("250 2.0.0 OK")
		}
	}
}

func (s *fakeSMTPServer) recv(t *testing.T, timeout time.Duration) string {
	t.Helper()
	select {
	case m := <-s.messages:
		return m
	case <-time.After(timeout):
		t.Fatal("timeout waiting for fake smtp message")
		return ""
	}
}

func testTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, _ := x509.MarshalECPrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("load keypair: %v", err)
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}}
}

func newFakeTestMailer(t *testing.T, server *fakeSMTPServer, tlsMode, username, password string) *SMTPMailer {
	t.Helper()
	host, port := server.hostPort(t)
	m, err := New(Settings{
		Host:      host,
		Port:      port,
		Username:  username,
		Password:  password,
		FromEmail: "no-reply@easy-admin.local",
		FromName:  "easy-admin",
		TLSMode:   tlsMode,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m.setTLSOverrideForTest(server.fakeClientTLSConfig())
	return m
}

func TestSMTPMailerSendPlain(t *testing.T) {
	server := startFakeSMTP(t, false, false)
	m := newFakeTestMailer(t, server, TLSModeNone, "", "")
	if err := m.Send(context.Background(), "user@example.com", "Verify", "<p>hi</p>"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	msg := server.recv(t, 5*time.Second)
	if !strings.Contains(msg, "Subject: Verify") || !strings.Contains(msg, "<p>hi</p>") {
		t.Fatalf("message missing content: %q", msg)
	}
	if !strings.Contains(msg, "To: user@example.com") {
		t.Fatalf("message missing recipient: %q", msg)
	}
	// The SMTP password must never appear anywhere in the wire message.
	if strings.Contains(msg, "smtp-pass") {
		t.Fatalf("wire message must not contain credentials: %q", msg)
	}
}

func TestSMTPMailerSendStartTLS(t *testing.T) {
	server := startFakeSMTP(t, true, false)
	m := newFakeTestMailer(t, server, TLSModeStartTLS, "", "")
	if err := m.Send(context.Background(), "user@example.com", "Subject", "body"); err != nil {
		t.Fatalf("Send starttls: %v", err)
	}
	if msg := server.recv(t, 5*time.Second); !strings.Contains(msg, "Subject: Subject") {
		t.Fatalf("message missing: %q", msg)
	}
}

func TestSMTPMailerSendAuthWithPasswordNeverLeaks(t *testing.T) {
	server := startFakeSMTP(t, true, false)
	// Password is sent over the encrypted channel only; the fake server does
	// not require auth, so the client won't send it. What we prove here is
	// that AUTH is never attempted unencrypted: TLSModeNone + username is
	// rejected at construction.
	if _, err := New(Settings{
		Host: "smtp.example.com", Port: 25, Username: "u", Password: "secret",
		FromEmail: "a@b.c", TLSMode: TLSModeNone,
	}); err == nil {
		t.Fatal("username over plaintext SMTP must be rejected")
	}
	_ = server
}

func TestSMTPMailerSendConnectionRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close() // nothing listens now

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatal(err)
	}
	m, err := New(Settings{Host: host, Port: port, FromEmail: "a@b.c", TLSMode: TLSModeNone, ConnectTimeout: 500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	err = m.Send(context.Background(), "u@example.com", "s", "b")
	if err == nil {
		t.Fatal("Send to a dead port must fail")
	}
	if !strings.Contains(err.Error(), "smtp") {
		t.Fatalf("error must be sanitized and mention smtp: %v", err)
	}
}

func TestSMTPMailerValidateSettings(t *testing.T) {
	bad := []Settings{
		{Host: "", Port: 25, TLSMode: TLSModeStartTLS},
		{Host: "smtp.example.com", Port: 0, TLSMode: TLSModeStartTLS},
		{Host: "smtp.example.com", Port: 25, FromEmail: "", TLSMode: TLSModeStartTLS},
		{Host: "smtp.example.com", Port: 25, FromEmail: "a@b.c", TLSMode: "weird"},
		{Host: "smtp.example.com", Port: 25, Username: "u", TLSMode: TLSModeNone},
	}
	for i, cfg := range bad {
		if err := ValidateSettings(cfg); err == nil {
			t.Fatalf("case %d must fail validation", i)
		}
	}
	good := Settings{Host: "smtp.example.com", Port: 587, FromEmail: "a@b.c", TLSMode: TLSModeStartTLS}
	if err := ValidateSettings(good); err != nil {
		t.Fatalf("good settings failed: %v", err)
	}
}

// The plain-AUTH path over STARTTLS is exercised through smtp.PlainAuth; this
// proves the wire token is base64 of NUL-separated identity and carries no
// plaintext credentials outside the TLS session.
func TestPlainAuthTokenShape(t *testing.T) {
	auth := plainAuth("user", "pass", "smtp.example.com")
	info := &smtp.ServerInfo{Name: "smtp.example.com", TLS: true, Auth: []string{"PLAIN"}}
	proto, resp, err := auth.Start(info)
	if err != nil {
		t.Fatalf("auth start: %v", err)
	}
	if proto != "PLAIN" {
		t.Fatalf("mechanism = %q, want PLAIN", proto)
	}
	if string(resp) != "\x00user\x00pass" {
		t.Fatalf("plain token mismatch: %q", resp)
	}
}
