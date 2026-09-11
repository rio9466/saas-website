package handler

import (
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/contact"
	"github.com/rio9466/easy-admin/server/internal/domain/media"
)

// --- public contact form ---------------------------------------------------

type publicContactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
	Message string `json:"message"`
	Locale  string `json:"locale"`
	Consent bool   `json:"consent"`
	// Website is the hidden honeypot; a non-empty value yields a silent 201.
	Website string `json:"website"`
}

func (r publicContactRequest) toInput() contact.SubmissionInput {
	return contact.SubmissionInput{
		Name:    r.Name,
		Email:   r.Email,
		Company: r.Company,
		Message: r.Message,
		Locale:  r.Locale,
		Consent: r.Consent,
		Website: r.Website,
	}
}

type contactReceiptData struct {
	ID          string `json:"id"`
	SubmittedAt string `json:"submitted_at"`
}

func toContactReceiptData(receipt *contact.Receipt) contactReceiptData {
	if receipt == nil {
		return contactReceiptData{}
	}
	return contactReceiptData{
		ID:          idString(receipt.ID),
		SubmittedAt: formatTime(receipt.SubmittedAt),
	}
}

// --- administrator contact inbox -------------------------------------------

type adminContactStatusRequest struct {
	Status string `json:"status"`
}

type contactSubmissionData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Company   string `json:"company"`
	Message   string `json:"message"`
	Locale    string `json:"locale"`
	Status    string `json:"status"`
	SourceIP  string `json:"source_ip"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toContactSubmissionData(in *contact.Submission) contactSubmissionData {
	if in == nil {
		return contactSubmissionData{}
	}
	return contactSubmissionData{
		ID:        idString(in.ID),
		Name:      in.Name,
		Email:     in.Email,
		Company:   in.Company,
		Message:   in.Message,
		Locale:    in.Locale,
		Status:    in.Status,
		SourceIP:  in.SourceIP,
		UserAgent: in.UserAgent,
		CreatedAt: formatTime(in.CreatedAt),
		UpdatedAt: formatTime(in.UpdatedAt),
	}
}

func toContactSubmissionList(in []contact.Submission) []contactSubmissionData {
	out := make([]contactSubmissionData, 0, len(in))
	for i := range in {
		out = append(out, toContactSubmissionData(&in[i]))
	}
	return out
}

// --- administrator media library -------------------------------------------

type mediaAssetData struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	MIME   string `json:"mime"`
	Size   int64  `json:"size"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func toMediaAssetData(in *media.Asset) mediaAssetData {
	if in == nil {
		return mediaAssetData{}
	}
	return mediaAssetData{
		ID:     idString(in.ID),
		URL:    in.URL,
		MIME:   in.MIME,
		Size:   in.SizeBytes,
		Width:  in.Width,
		Height: in.Height,
	}
}

func toMediaAssetList(in []media.Asset) []mediaAssetData {
	out := make([]mediaAssetData, 0, len(in))
	for i := range in {
		out = append(out, toMediaAssetData(&in[i]))
	}
	return out
}

// normalizeStatus trims a status query/body value.
func normalizeStatus(raw string) string {
	return strings.TrimSpace(raw)
}
