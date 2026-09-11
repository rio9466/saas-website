package http_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	transporthttp "github.com/rio9466/easy-admin/server/internal/transport/http"
)

// parseOpenAPI loads docs/openapi.yaml into a generic tree.
func parseOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	var root map[string]any
	if err := yaml.Unmarshal([]byte(readOpenAPI(t)), &root); err != nil {
		t.Fatalf("openapi.yaml is not valid YAML: %v", err)
	}
	if root["openapi"] != "3.0.3" {
		t.Fatalf("openapi version = %v, want 3.0.3", root["openapi"])
	}
	if _, ok := root["paths"]; !ok {
		t.Fatal("openapi.yaml missing paths")
	}
	return root
}

func anyMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func anySlice(v any) []any {
	s, _ := v.([]any)
	return s
}

// TestOpenAPIAndRouterFullParity proves every registered HTTP operation is
// documented and every documented operation is registered, for health probes
// and every admin operation.
func TestOpenAPIAndRouterFullParity(t *testing.T) {
	t.Parallel()

	root := parseOpenAPI(t)
	documented := map[string]map[string]struct{}{}
	for path, node := range anyMap(root["paths"]) {
		for method := range anyMap(node) {
			lm := strings.ToLower(method)
			if lm == "parameters" || lm == "summary" || lm == "description" {
				continue
			}
			if documented[path] == nil {
				documented[path] = map[string]struct{}{}
			}
			documented[path][lm] = struct{}{}
		}
	}

	router := transporthttp.NewRouter(transporthttp.Dependencies{
		ReadyChecker: stubChecker{},
		AdminAuth:    &stubAdminAuth{},
		Tokens:       stubTokens{},
		Sessions:     stubSessions{},
		UserClient:   &stubUserClient{},
		UserAdmin:    &stubUserAdmin{},
		UserTokens:   stubTokens{},
		UserSessions: stubSessions{},
	})
	registered := map[string]map[string]struct{}{}
	for _, route := range router.Routes() {
		path := normalizeGinPath(route.Path)
		method := strings.ToLower(route.Method)
		if method == "head" || method == "options" {
			// Gin synthesizes HEAD for GET routes; the contract is GET.
			continue
		}
		if registered[path] == nil {
			registered[path] = map[string]struct{}{}
		}
		registered[path][method] = struct{}{}
	}

	if len(documented) == 0 {
		t.Fatal("documented operation set is empty")
	}
	for path, methods := range documented {
		rm, ok := registered[path]
		if !ok {
			t.Fatalf("router does not register documented path %s", path)
		}
		for method := range methods {
			if _, ok := rm[method]; !ok {
				t.Fatalf("router does not register documented operation %s %s", strings.ToUpper(method), path)
			}
		}
	}
	for path, methods := range registered {
		dm, ok := documented[path]
		if !ok {
			t.Fatalf("openapi.yaml does not document registered path %s", path)
		}
		for method := range methods {
			if _, ok := dm[method]; !ok {
				t.Fatalf("openapi.yaml does not document registered operation %s %s", strings.ToUpper(method), path)
			}
		}
	}
}

// TestOpenAPIReferencesResolve walks every $ref in openapi.yaml and proves its
// target exists under components.
func TestOpenAPIReferencesResolve(t *testing.T) {
	t.Parallel()

	root := parseOpenAPI(t)
	components := anyMap(root["components"])
	if components == nil {
		t.Fatal("openapi.yaml missing components")
	}
	var missing []string
	var walk func(v any)
	walk = func(v any) {
		switch node := v.(type) {
		case map[string]any:
			if ref, ok := node["$ref"].(string); ok && strings.HasPrefix(ref, "#/") {
				if !resolvePointer(root, ref) {
					missing = append(missing, ref)
				}
				return
			}
			for _, child := range node {
				walk(child)
			}
		case []any:
			for _, child := range node {
				walk(child)
			}
		}
	}
	walk(components)
	if len(missing) > 0 {
		t.Fatalf("unresolved $refs: %v", missing)
	}
	if !resolvePointer(root, "#/components/schemas/Envelope") {
		t.Fatal("sanity: Envelope ref failed to resolve")
	}
}

func resolvePointer(root map[string]any, ref string) bool {
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	var cur any = root
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		cur, ok = m[part]
		if !ok {
			return false
		}
	}
	return true
}

func operationByID(t *testing.T, root map[string]any, operationID string) map[string]any {
	t.Helper()
	for _, pathNode := range anyMap(root["paths"]) {
		for _, methodNode := range anyMap(pathNode) {
			if methodNode == nil {
				continue
			}
			op := anyMap(methodNode)
			if op["operationId"] == operationID {
				return op
			}
		}
	}
	t.Fatalf("operation %s not found in openapi.yaml", operationID)
	return nil
}

func requiredFields(t *testing.T, op map[string]any) []string {
	t.Helper()
	rb := anyMap(op["requestBody"])
	if rb == nil {
		t.Fatalf("operation has no request body")
	}
	content := anyMap(rb["content"])
	json := anyMap(content["application/json"])
	schema := anyMap(json["schema"])
	var out []string
	for _, v := range anySlice(schema["required"]) {
		out = append(out, fmt.Sprint(v))
	}
	return out
}

func assertRequired(t *testing.T, op map[string]any, want ...string) {
	t.Helper()
	got := requiredFields(t, op)
	for _, field := range want {
		found := false
		for _, g := range got {
			if g == field {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("operation %s request body missing required field %q (have %v)", op["operationId"], field, got)
		}
	}
}

func assertNotRequired(t *testing.T, op map[string]any, forbidden ...string) {
	t.Helper()
	got := requiredFields(t, op)
	for _, field := range forbidden {
		for _, g := range got {
			if g == field {
				t.Fatalf("operation %s request body must not require legacy field %q", op["operationId"], field)
			}
		}
	}
}

// TestOpenAPIAuthContractFields locks the wire fields that the handlers bind:
// reset-password uses new_password (not password), login/me/password bodies,
// role assignment bodies, create status 201, and the login token payload
// including token_type.
func TestOpenAPIAuthContractFields(t *testing.T) {
	t.Parallel()

	root := parseOpenAPI(t)

	reset := operationByID(t, root, "resetAdministratorPassword")
	assertRequired(t, reset, "new_password")
	assertNotRequired(t, reset, "password")

	login := operationByID(t, root, "adminLogin")
	assertRequired(t, login, "username", "password")

	change := operationByID(t, root, "adminChangePassword")
	assertRequired(t, change, "current_password", "new_password")

	createAdmin := operationByID(t, root, "createAdministrator")
	assertRequired(t, createAdmin, "username", "password", "role_codes")
	assertHasStatus(t, createAdmin, "201")
	assertNotStatus(t, createAdmin, "200")

	createRole := operationByID(t, root, "createRole")
	assertHasStatus(t, createRole, "201")

	assign := operationByID(t, root, "assignAdministratorRoles")
	assertRequired(t, assign, "role_codes")

	replacePerms := operationByID(t, root, "replaceRolePermissions")
	assertRequired(t, replacePerms, "permission_codes")

	// LoginSuccess documents access_token, token_type, and expires_in.
	loginSuccess := anyMap(anyMap(anyMap(root["components"])["schemas"])["LoginSuccess"])
	var dataShape map[string]any
	for _, item := range anySlice(loginSuccess["allOf"]) {
		member := anyMap(item)
		props := anyMap(member["properties"])
		if data := anyMap(props["data"]); data != nil {
			dataShape = data
			break
		}
	}
	if dataShape == nil {
		t.Fatal("LoginSuccess data shape not found")
	}
	propSet := map[string]struct{}{}
	for key := range anyMap(dataShape["properties"]) {
		propSet[key] = struct{}{}
	}
	for _, want := range []string{"access_token", "token_type", "expires_in"} {
		if _, ok := propSet[want]; !ok {
			t.Fatalf("LoginSuccess.data missing property %q", want)
		}
	}
	for _, want := range []string{"access_token", "token_type", "expires_in"} {
		found := false
		for _, r := range anySlice(dataShape["required"]) {
			if fmt.Sprint(r) == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("LoginSuccess.data required fields missing %q (have %v)", want, dataShape["required"])
		}
	}
}

func assertHasStatus(t *testing.T, op map[string]any, status string) {
	t.Helper()
	if _, ok := anyMap(op["responses"])[status]; !ok {
		t.Fatalf("operation %s responses missing %s: %v", op["operationId"], status, op["responses"])
	}
}

func assertNotStatus(t *testing.T, op map[string]any, status string) {
	t.Helper()
	if _, ok := anyMap(op["responses"])[status]; ok {
		t.Fatalf("operation %s must not document %s after aligning statuses", op["operationId"], status)
	}
}
