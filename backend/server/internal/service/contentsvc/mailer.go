package contentsvc

import (
	"context"
	"fmt"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/contact"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/platform/mailer"
)

// notifyContactSubmission best-effort delivers a notification email to the
// configured site contact address. Every failure is logged at error level and
// never propagated: the submission is already persisted, so the public request
// must still succeed.
func (s *Service) notifyContactSubmission(ctx context.Context, submission *contact.Submission) {
	if submission == nil || s.users == nil {
		return
	}
	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		s.logger.Error("contact notification email failed",
			"submission_id", idString(submission.ID), "reason", "load_system_settings", "error", err)
		return
	}
	if !settings.SMTPEnabled {
		return
	}
	siteSettings, err := s.content.GetSiteSettings(ctx)
	if err != nil {
		s.logger.Error("contact notification email failed",
			"submission_id", idString(submission.ID), "reason", "load_site_settings", "error", err)
		return
	}
	recipient := strings.TrimSpace(siteSettings.ContactEmail)
	if recipient == "" {
		s.logger.Warn("contact notification email skipped: no contact email configured",
			"submission_id", idString(submission.ID))
		return
	}

	plain, err := s.storedSMTPPassword(ctx, settings)
	if err != nil {
		s.logger.Error("contact notification email failed",
			"submission_id", idString(submission.ID), "reason", "smtp_password", "error", err)
		return
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
		s.logger.Error("contact notification email failed",
			"submission_id", idString(submission.ID), "reason", "configure_mailer", "error", err)
		return
	}
	subject := fmt.Sprintf("[%s] New contact message from %s", settings.PlatformName, submission.Name)
	if err := m.Send(ctx, recipient, subject, contactEmailBody(submission)); err != nil {
		s.logger.Error("contact notification email failed",
			"submission_id", idString(submission.ID), "reason", "send", "error", err)
	}
}

// storedSMTPPassword decrypts the stored SMTP password only when a master key
// is configured. No plaintext or ciphertext is ever logged.
func (s *Service) storedSMTPPassword(ctx context.Context, settings *userdomain.SystemSettings) (string, error) {
	if settings.SMTPUsername == "" {
		return "", nil
	}
	if s.box == nil {
		return "", fmt.Errorf("smtp master key is not configured")
	}
	ciphertext, err := s.users.GetSystemSettingsCiphertext(ctx)
	if err != nil {
		return "", err
	}
	plain, err := s.box.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("stored smtp password cannot be decrypted")
	}
	return plain, nil
}

func contactEmailBody(submission *contact.Submission) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><body style=\"font-family:sans-serif;max-width:620px;margin:0 auto\">")
	b.WriteString("<h2>New contact message</h2>")
	b.WriteString("<table cellpadding=\"4\">")
	writeContactRow(&b, "Name", submission.Name)
	writeContactRow(&b, "Email", submission.Email)
	if submission.Company != "" {
		writeContactRow(&b, "Company", submission.Company)
	}
	if submission.Locale != "" {
		writeContactRow(&b, "Locale", submission.Locale)
	}
	b.WriteString("</table>")
	b.WriteString("<p style=\"white-space:pre-wrap\">")
	b.WriteString(escapeHTML(submission.Message))
	b.WriteString("</p>")
	b.WriteString("</body></html>")
	return b.String()
}

func writeContactRow(b *strings.Builder, label, value string) {
	b.WriteString("<tr><td><strong>")
	b.WriteString(escapeHTML(label))
	b.WriteString("</strong></td><td>")
	b.WriteString(escapeHTML(value))
	b.WriteString("</td></tr>")
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return replacer.Replace(s)
}
