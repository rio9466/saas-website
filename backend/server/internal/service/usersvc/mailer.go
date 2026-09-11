package usersvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/platform/mailer"
)

// Verification token Redis keys. Tokens are stored hashed (sha256) so a Redis
// leak never exposes a usable verification link.
const (
	userVerifyKeyPrefix = "easy-admin:user-verify:"
	// Verification token entropy: 32 random bytes rendered as 64 hex chars.
	verifyTokenBytes = 32
)

var (
	ErrVerificationInvalid = errors.New("verification token invalid")
)

// generateVerificationToken returns a high-entropy one-time token string.
func generateVerificationToken() (string, error) {
	var buf [verifyTokenBytes]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate verification token: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

// hashVerificationToken stores only the sha256 digest.
func hashVerificationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func userVerifyKey(suffix string) string {
	return userVerifyKeyPrefix + suffix
}

// storeVerificationToken persists the hash with a bounded TTL. A failure here
// must fail closed: the caller returns a 503 instead of risking a token that
// cannot be validated.
func (s *Service) storeVerificationToken(ctx context.Context, userID int64, tokenHash string) error {
	if s.sessions == nil {
		return ErrVerificationInvalid
	}
	ttl := time.Duration(s.verifyTTL)
	if err := s.sessions.Raw().Set(ctx, userVerifyKey(idString(userID)), tokenHash, ttl).Err(); err != nil {
		return fmt.Errorf("store verification token: %w", err)
	}
	return nil
}

// takeVerificationToken atomically consumes a token (one-time use). A Lua
// script compares the stored hash and deletes only on an exact match, so a
// wrong or replayed token is rejected without burning the still-valid one.
// Returns (false, nil) when no token is stored or the hashes differ; Redis
// errors fail closed.
func (s *Service) takeVerificationToken(ctx context.Context, userID int64, token string) (bool, error) {
	if s.sessions == nil {
		return false, ErrVerificationInvalid
	}
	key := userVerifyKey(idString(userID))
	expected := hashVerificationToken(token)
	script := goredis.NewScript(`
local v = redis.call('GET', KEYS[1])
if not v then
  return 0
end
if v == ARGV[1] then
  redis.call('DEL', KEYS[1])
  return 1
end
return 2
`)
	res, err := script.Run(ctx, s.sessions.Raw(), []string{key}, expected).Int64()
	if err != nil {
		return false, fmt.Errorf("redeem verification token: %w", err)
	}
	return res == 1, nil
}

// sendVerificationEmail builds the mailer from live system settings and sends
// the verification link. Failures are sanitized; the SMTP password never
// appears. The mailer is constructed per send so settings rotation takes
// effect immediately.
func (s *Service) sendVerificationEmail(ctx context.Context, settings *userdomain.SystemSettings, toEmail, token string) error {
	link := s.verificationLink(settings.PublicFrontendURL, token)
	subject := fmt.Sprintf("%s: verify your email address", settings.PlatformName)
	body := verificationEmailBody(settings.PlatformName, link, s.verifyTTL)

	plain, err := s.storedSMTPPassword(ctx, settings)
	if err != nil {
		return err
	}
	m, err := s.newMailer(mailer.Settings{
		Host:      settings.SMTPHost,
		Port:      settings.SMTPPort,
		Username:  settings.SMTPUsername,
		Password:  plain,
		FromEmail: settings.SMTPFromEmail,
		FromName:  settings.SMTPFromName,
		TLSMode:   settings.SMTPTLSMode,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", apperr.ErrSmtpUnavailable, err)
	}
	if err := m.Send(ctx, toEmail, subject, body); err != nil {
		return fmt.Errorf("%w: %v", apperr.ErrSmtpUnavailable, err)
	}
	return nil
}

// storedSMTPPassword decrypts the stored SMTP password only when a master key
// is configured. No plaintext or ciphertext is ever logged.
func (s *Service) storedSMTPPassword(ctx context.Context, settings *userdomain.SystemSettings) (string, error) {
	if settings.SMTPUsername == "" {
		return "", nil
	}
	if s.box == nil {
		return "", fmt.Errorf("%w: smtp master key is not configured", apperr.ErrSmtpUnavailable)
	}
	ciphertext, err := s.users.GetSystemSettingsCiphertext(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %v", apperr.ErrSmtpUnavailable, err)
	}
	plain, err := s.box.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: stored smtp password cannot be decrypted", apperr.ErrSmtpUnavailable)
	}
	return plain, nil
}

func (s *Service) verificationLink(frontendURL, token string) string {
	base := strings.TrimRight(strings.TrimSpace(frontendURL), "/")
	if base == "" {
		base = "/"
	}
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/verify-email"
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String()
}

func verificationEmailBody(platformName, link string, ttlNanos int64) string {
	hours := int64(24)
	if ttlNanos > 0 {
		hours = ttlNanos / int64(time.Hour)
		if hours < 1 {
			hours = 1
		}
	}
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><body style=\"font-family:sans-serif;max-width:560px;margin:0 auto\">")
	b.WriteString("<h2>")
	b.WriteString(escapeHTML(platformName))
	b.WriteString("</h2>")
	b.WriteString("<p>Welcome! Please confirm your email address to finish setting up your account.</p>")
	b.WriteString("<p><a href=\"")
	b.WriteString(escapeHTMLAttr(link))
	b.WriteString("\">Verify my email address</a></p>")
	b.WriteString("<p>This link is single-use and expires after ")
	b.WriteString(fmt.Sprintf("%d", hours))
	b.WriteString(" hours. If you did not register, you can safely ignore this message.</p>")
	b.WriteString("</body></html>")
	return b.String()
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return replacer.Replace(s)
}

func escapeHTMLAttr(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return replacer.Replace(s)
}
