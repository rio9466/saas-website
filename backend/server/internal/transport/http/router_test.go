package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	transporthttp "github.com/rio9466/easy-admin/server/internal/transport/http"
	"github.com/rio9466/easy-admin/server/internal/transport/http/handler"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

type stubChecker struct {
	err    error
	called *bool
}

func (s stubChecker) Ping(context.Context) error {
	if s.called != nil {
		*s.called = true
	}
	return s.err
}

func TestHealthzDoesNotCallDependency(t *testing.T) {
	t.Parallel()

	calledPrimary := false
	calledLog := false
	calledRedis := false

	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: handler.MultiReady(slog.Default(),
			handler.NamedReadyCheck{Name: "primary_postgres", Checker: stubChecker{called: &calledPrimary}},
			handler.NamedReadyCheck{Name: "log_postgres", Checker: stubChecker{called: &calledLog}},
			handler.NamedReadyCheck{Name: "redis", Checker: stubChecker{called: &calledRedis}},
		),
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if calledPrimary || calledLog || calledRedis {
		t.Fatal("healthz must not call readiness dependencies")
	}
	assertSuccess(t, rec, http.StatusOK, map[string]string{"status": "ok"})
}

func TestReadyzAllDependenciesSucceed(t *testing.T) {
	t.Parallel()

	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: handler.MultiReady(nil,
			handler.NamedReadyCheck{Name: "primary_postgres", Checker: stubChecker{}, Timeout: time.Second},
			handler.NamedReadyCheck{Name: "log_postgres", Checker: stubChecker{}, Timeout: time.Second},
			handler.NamedReadyCheck{Name: "redis", Checker: stubChecker{}, Timeout: time.Second},
		),
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	req.Header.Set(middleware.HeaderRequestID, "req-fixed-001")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assertSuccess(t, rec, http.StatusOK, map[string]string{"status": "ready"})
	if got := rec.Header().Get(middleware.HeaderRequestID); got != "req-fixed-001" {
		t.Fatalf("X-Request-ID = %q, want req-fixed-001", got)
	}
}

func TestReadyzEachDependencyFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		primary  error
		logDB    error
		redis    error
		wantCode int
	}{
		{name: "primary fails", primary: errors.New("primary down"), wantCode: http.StatusServiceUnavailable},
		{name: "log fails", logDB: errors.New("log down"), wantCode: http.StatusServiceUnavailable},
		{name: "redis fails", redis: errors.New("redis down"), wantCode: http.StatusServiceUnavailable},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := transporthttp.NewRouter(transporthttp.Dependencies{
				ReadyChecker: handler.MultiReady(nil,
					handler.NamedReadyCheck{Name: "primary_postgres", Checker: stubChecker{err: tt.primary}},
					handler.NamedReadyCheck{Name: "log_postgres", Checker: stubChecker{err: tt.logDB}},
					handler.NamedReadyCheck{Name: "redis", Checker: stubChecker{err: tt.redis}},
				),
			})

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			req.Header.Set(middleware.HeaderRequestID, "req-fail-001")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assertReadyzFailure(t, rec, "req-fail-001")
			body := rec.Body.String()
			if strings.Contains(body, "primary") || strings.Contains(body, "redis") || strings.Contains(body, "log down") {
				t.Fatalf("response leaked dependency details: %s", body)
			}
		})
	}
}

func TestRequestIDGeneratedWhenMissing(t *testing.T) {
	t.Parallel()

	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: handler.MultiReady(nil, handler.NamedReadyCheck{Name: "primary_postgres", Checker: stubChecker{}}),
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	headerID := rec.Header().Get(middleware.HeaderRequestID)
	if headerID == "" {
		t.Fatal("expected generated X-Request-ID header")
	}
	if !middleware.ValidRequestID(headerID) {
		t.Fatalf("generated request ID %q is invalid", headerID)
	}

	var env response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.RequestID != headerID {
		t.Fatalf("body request_id = %q, header = %q", env.RequestID, headerID)
	}
}

func TestRequestIDValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		inbound   string
		wantEcho  bool
		wantExact string
	}{
		{name: "valid echoed", inbound: "req-fixed_001:v2", wantEcho: true, wantExact: "req-fixed_001:v2"},
		{name: "empty generates", inbound: "", wantEcho: false},
		{name: "oversized generates", inbound: strings.Repeat("a", middleware.MaxRequestIDLength+1), wantEcho: false},
		{name: "whitespace generates", inbound: "req 001", wantEcho: false},
		{name: "unicode generates", inbound: "req-请求-001", wantEcho: false},
		{name: "invalid punctuation generates", inbound: "req/001", wantEcho: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := transporthttp.NewRouter(transporthttp.Dependencies{
				ReadyChecker: stubChecker{},
			})

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			if tt.inbound != "" {
				req.Header.Set(middleware.HeaderRequestID, tt.inbound)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			headerID := rec.Header().Get(middleware.HeaderRequestID)
			if !middleware.ValidRequestID(headerID) {
				t.Fatalf("response header request ID %q is invalid", headerID)
			}

			var env response.Envelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("decode envelope: %v", err)
			}
			if env.RequestID != headerID {
				t.Fatalf("body request_id = %q, header = %q", env.RequestID, headerID)
			}

			if tt.wantEcho {
				if headerID != tt.wantExact {
					t.Fatalf("X-Request-ID = %q, want echoed %q", headerID, tt.wantExact)
				}
				return
			}
			if headerID == tt.inbound {
				t.Fatalf("invalid inbound %q was echoed instead of regenerated", tt.inbound)
			}
		})
	}
}

func assertReadyzFailure(t *testing.T, rec *httptest.ResponseRecorder, requestID string) {
	t.Helper()
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	var env response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.Code != handler.CodeDependencyUnavailable {
		t.Fatalf("code = %d, want %d", env.Code, handler.CodeDependencyUnavailable)
	}
	if env.Message != "dependency unavailable" {
		t.Fatalf("message = %q, want dependency unavailable", env.Message)
	}
	if env.RequestID != requestID {
		t.Fatalf("request_id = %q, want %q", env.RequestID, requestID)
	}
}

func assertSuccess(t *testing.T, rec *httptest.ResponseRecorder, status int, data map[string]string) {
	t.Helper()

	if rec.Code != status {
		t.Fatalf("status = %d, want %d", rec.Code, status)
	}

	var env response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.Code != 0 {
		t.Fatalf("code = %d, want 0", env.Code)
	}
	if env.Message != "success" {
		t.Fatalf("message = %q, want success", env.Message)
	}
	if env.RequestID == "" {
		t.Fatal("request_id is empty")
	}
	if rec.Header().Get(middleware.HeaderRequestID) != env.RequestID {
		t.Fatalf("header request id %q != body request id %q",
			rec.Header().Get(middleware.HeaderRequestID), env.RequestID)
	}

	raw, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	for k, want := range data {
		if got[k] != want {
			t.Fatalf("data[%q] = %q, want %q", k, got[k], want)
		}
	}
}
