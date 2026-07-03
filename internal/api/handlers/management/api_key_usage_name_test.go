package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// TestGetAPIKeyUsage_IncludesNameAttribute asserts that a non-OpenAI auth
// carrying a "name" attribute surfaces it in the usage entry, so the panel can
// distinguish multiple accounts of the same provider by name.
func TestGetAPIKeyUsage_IncludesNameAttribute(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "")

	manager := coreauth.NewManager(nil, nil, nil)
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:       "gemini-auth",
		Provider: "gemini",
		Attributes: map[string]string{
			"api_key":  "gemini-key",
			"base_url": "https://gemini.example.com",
			"name":     "my-google-acct",
		},
	}); err != nil {
		t.Fatalf("register gemini auth: %v", err)
	}
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:       "gemini-auth-2",
		Provider: "gemini",
		Attributes: map[string]string{
			"api_key":  "gemini-key-2",
			"base_url": "https://gemini-2.example.com",
			// no name -> field omitted (omitempty)
		},
	}); err != nil {
		t.Fatalf("register gemini auth 2: %v", err)
	}

	manager.MarkResult(context.Background(), coreauth.Result{AuthID: "gemini-auth", Provider: "gemini", Model: "gemini-3-pro", Success: true})

	h := NewHandlerWithoutConfigFilePath(&config.Config{AuthDir: t.TempDir()}, manager)

	rec := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(rec)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/v0/management/api-key-usage", nil)
	h.GetAPIKeyUsage(ginCtx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload map[string]map[string]apiKeyUsageEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	geminiBucket := payload["gemini"]
	if geminiBucket == nil {
		t.Fatalf("missing gemini provider bucket: %#v", payload)
	}
	named := geminiBucket["https://gemini.example.com|gemini-key"]
	if named.Name != "my-google-acct" {
		t.Fatalf("name = %q, want %q", named.Name, "my-google-acct")
	}
	unnamed := geminiBucket["https://gemini-2.example.com|gemini-key-2"]
	if unnamed.Name != "" {
		t.Fatalf("unnamed entry name = %q, want empty (omitempty)", unnamed.Name)
	}
}
