package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/domain/contact"
	"github.com/rio9466/easy-admin/server/internal/domain/media"
)

// stubInboxService implements only the contact/media methods under test; the
// embedded nil interface satisfies the rest of ContentService.
type stubInboxService struct {
	ContentService
	receipt   *contact.Receipt
	submitErr error
	asset     *media.Asset
	uploadErr error
}

func (s *stubInboxService) SubmitContact(context.Context, contact.SubmissionInput, string, string) (*contact.Receipt, error) {
	return s.receipt, s.submitErr
}

func (s *stubInboxService) UploadMedia(context.Context, Actor, string, int64, io.Reader) (*media.Asset, error) {
	return s.asset, s.uploadErr
}

func newInboxTestRouter(svc ContentService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	public := NewContentPublicHandlers(svc)
	r.POST("/api/v1/public/contact", public.Contact)
	admin := NewContentAdminHandlers(svc)
	r.POST("/api/v1/admin/media", admin.UploadMedia)
	return r
}

type testEnvelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func TestPublicContactReturns201NoStore(t *testing.T) {
	t.Parallel()
	svc := &stubInboxService{receipt: &contact.Receipt{ID: 12}}
	r := newInboxTestRouter(svc)

	body := `{"name":"A","email":"a@b.com","message":"hi","consent":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	var env testEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Code != 0 {
		t.Fatalf("code = %d, want 0", env.Code)
	}
	var data struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if data.ID != "12" {
		t.Fatalf("data.id = %q, want 12", data.ID)
	}
}

func TestPublicContactMapsValidationError(t *testing.T) {
	t.Parallel()
	svc := &stubInboxService{submitErr: apperr.New(http.StatusBadRequest, apperr.CodeValidation, "validation failed", apperr.ErrValidation)}
	r := newInboxTestRouter(svc)

	body := `{"name":"A","email":"a@b.com","message":"hi","consent":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/public/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
	}
	var env testEnvelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != apperr.CodeValidation {
		t.Fatalf("code = %d, want 10001", env.Code)
	}
}

func TestUploadMediaMultipartReturns201(t *testing.T) {
	t.Parallel()
	svc := &stubInboxService{asset: &media.Asset{ID: 7, URL: "/media/x.png", MIME: media.MIMEPNG, SizeBytes: 3, Width: 3, Height: 2}}
	r := newInboxTestRouter(svc)

	req := multipartUploadRequest(t, http.MethodPost, "/api/v1/admin/media", "x.png", []byte("png"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	var env testEnvelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	var data struct {
		URL  string `json:"url"`
		Size int64  `json:"size"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if data.URL != "/media/x.png" || data.Size != 3 {
		t.Fatalf("data = %+v, want url=/media/x.png size=3", data)
	}
}

func TestUploadMediaRejectsUnsupportedType(t *testing.T) {
	t.Parallel()
	svc := &stubInboxService{uploadErr: apperr.New(http.StatusBadRequest, apperr.CodeValidation, "validation failed", apperr.ErrValidation)}
	r := newInboxTestRouter(svc)

	req := multipartUploadRequest(t, http.MethodPost, "/api/v1/admin/media", "x.txt", []byte("hello"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", w.Code, w.Body.String())
	}
	var env testEnvelope
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != apperr.CodeValidation {
		t.Fatalf("code = %d, want 10001", env.Code)
	}
}

func multipartUploadRequest(t *testing.T, method, target, filename string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
