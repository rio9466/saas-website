package http_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	transporthttp "github.com/rio9466/easy-admin/server/internal/transport/http"
)

func TestOpenAPIAdminRoutesMatchRouter(t *testing.T) {
	t.Parallel()

	openapi := readOpenAPI(t)
	assertOpenAPIHas(t, openapi, "get:", "/api/v1/admin/administrators/{id}")
	assertOpenAPICreateUses201(t, openapi)

	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: stubChecker{},
		AdminAuth:    &stubAdminAuth{},
		Tokens:       stubTokens{},
		Sessions:     stubSessions{},
	})

	registered := map[string]map[string]struct{}{}
	for _, route := range router.Routes() {
		path := normalizeGinPath(route.Path)
		if registered[path] == nil {
			registered[path] = map[string]struct{}{}
		}
		registered[path][strings.ToUpper(route.Method)] = struct{}{}
	}

	required := []struct {
		path   string
		method string
	}{
		{"/healthz", "GET"},
		{"/readyz", "GET"},
		{"/api/v1/admin/auth/login", "POST"},
		{"/api/v1/admin/auth/refresh", "POST"},
		{"/api/v1/admin/auth/logout", "POST"},
		{"/api/v1/admin/me", "GET"},
		{"/api/v1/admin/me", "PATCH"},
		{"/api/v1/admin/me/password", "POST"},
		{"/api/v1/admin/administrators", "GET"},
		{"/api/v1/admin/administrators", "POST"},
		{"/api/v1/admin/administrators/{id}", "GET"},
		{"/api/v1/admin/administrators/{id}", "PATCH"},
		{"/api/v1/admin/administrators/{id}/enable", "POST"},
		{"/api/v1/admin/administrators/{id}/disable", "POST"},
		{"/api/v1/admin/administrators/{id}/reset-password", "POST"},
		{"/api/v1/admin/administrators/{id}/roles", "PUT"},
		{"/api/v1/admin/roles", "GET"},
		{"/api/v1/admin/roles", "POST"},
		{"/api/v1/admin/roles/{id}", "GET"},
		{"/api/v1/admin/roles/{id}", "PATCH"},
		{"/api/v1/admin/roles/{id}/permissions", "PUT"},
		{"/api/v1/admin/permissions", "GET"},
		{"/api/v1/admin/audit-events", "GET"},
		{"/api/v1/admin/audit-events/{id}", "GET"},
	}

	for _, want := range required {
		methods, ok := registered[want.path]
		if !ok {
			t.Fatalf("router missing path %s", want.path)
		}
		if _, ok := methods[want.method]; !ok {
			t.Fatalf("router missing %s %s (have %v)", want.method, want.path, keys(methods))
		}
		if !strings.Contains(openapi, want.path) {
			t.Fatalf("openapi missing path %s", want.path)
		}
	}
}

func readOpenAPI(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "docs", "openapi.yaml"))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}
	return string(raw)
}

func assertOpenAPIHas(t *testing.T, openapi, methodKey, path string) {
	t.Helper()
	idx := strings.Index(openapi, "  "+path+":")
	if idx < 0 {
		t.Fatalf("openapi missing %s", path)
	}
	section := openapi[idx:]
	if end := strings.Index(section[len(path)+4:], "\n  /"); end >= 0 {
		section = section[:len(path)+4+end]
	}
	if !strings.Contains(section, "\n    "+methodKey) && !strings.Contains(section, "\n    "+strings.TrimSuffix(methodKey, ":")+":") {
		// methodKey like "get:"
		if !strings.Contains(section, "\n    get:") {
			t.Fatalf("openapi path %s missing method %s\nsection:\n%s", path, methodKey, section)
		}
	}
}

func assertOpenAPICreateUses201(t *testing.T, openapi string) {
	t.Helper()
	marker := "operationId: createAdministrator"
	idx := strings.Index(openapi, marker)
	if idx < 0 {
		t.Fatal("createAdministrator missing")
	}
	section := openapi[idx:]
	if end := strings.Index(section, "\n  /api/v1/admin/administrators/{id}:"); end > 0 {
		section = section[:end]
	}
	if !strings.Contains(section, "'201':") && !strings.Contains(section, `"201":`) {
		t.Fatalf("createAdministrator responses must include 201\n%s", section)
	}
	if strings.Contains(section, "'200':") {
		t.Fatal("createAdministrator must not keep 200 after aligning to 201")
	}
}

func normalizeGinPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
