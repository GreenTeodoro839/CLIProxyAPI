package management

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

// TestPatchGeminiKey_ByName updates an entry located by name (not index/api-key).
func TestPatchGeminiKey_ByName(t *testing.T) {
	t.Parallel()

	h := &Handler{
		cfg: &config.Config{
			GeminiKey: []config.GeminiKey{
				{Name: "acct-a", APIKey: "key-a", BaseURL: "https://a.example.com"},
				{Name: "acct-b", APIKey: "key-b", BaseURL: "https://b.example.com"},
			},
		},
		configFilePath: writeTestConfigFile(t),
	}

	body := `{"name":"acct-a","value":{"base-url":"https://new.example.com","name":"acct-a-renamed"}}`
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/gemini-api-key", strings.NewReader(body))

	h.PatchGeminiKey(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := h.cfg.GeminiKey[0].BaseURL; got != "https://new.example.com" {
		t.Fatalf("patched base-url = %q, want %q", got, "https://new.example.com")
	}
	if got := h.cfg.GeminiKey[0].Name; got != "acct-a-renamed" {
		t.Fatalf("patched name = %q, want %q", got, "acct-a-renamed")
	}
	if got := h.cfg.GeminiKey[1].BaseURL; got != "https://b.example.com" {
		t.Fatalf("unmatched entry base-url changed unexpectedly = %q", got)
	}
}

// TestPatchClaudeKey_ByName_NotFound returns 404 when no entry matches the name.
func TestPatchClaudeKey_ByName_NotFound(t *testing.T) {
	t.Parallel()

	h := &Handler{
		cfg: &config.Config{
			ClaudeKey: []config.ClaudeKey{
				{Name: "acct-a", APIKey: "key-a"},
			},
		},
		configFilePath: writeTestConfigFile(t),
	}

	body := `{"name":"missing","value":{"base-url":"https://x.example.com"}}`
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPatch, "/v0/management/claude-api-key", strings.NewReader(body))

	h.PatchClaudeKey(c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if got := h.cfg.ClaudeKey[0].BaseURL; got != "" {
		t.Fatalf("base-url changed on unmatched entry = %q", got)
	}
}

// TestDeleteGeminiKey_ByName removes ALL entries matching the name.
func TestDeleteGeminiKey_ByName(t *testing.T) {
	t.Parallel()

	h := &Handler{
		cfg: &config.Config{
			GeminiKey: []config.GeminiKey{
				{Name: "acct-a", APIKey: "key-a", BaseURL: "https://a.example.com"},
				{Name: "acct-b", APIKey: "key-b", BaseURL: "https://b.example.com"},
				{Name: "acct-a", APIKey: "key-c", BaseURL: "https://c.example.com"},
			},
		},
		configFilePath: writeTestConfigFile(t),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodDelete, "/v0/management/gemini-api-key?name=acct-a", nil)

	h.DeleteGeminiKey(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := len(h.cfg.GeminiKey); got != 1 {
		t.Fatalf("gemini keys len = %d, want 1 (all acct-a removed)", got)
	}
	if h.cfg.GeminiKey[0].Name != "acct-b" {
		t.Fatalf("remaining name = %q, want acct-b", h.cfg.GeminiKey[0].Name)
	}
}

// TestDeleteVertexCompatKey_ByName_NotFound leaves config unchanged.
func TestDeleteVertexCompatKey_ByName_NotFound(t *testing.T) {
	t.Parallel()

	h := &Handler{
		cfg: &config.Config{
			VertexCompatAPIKey: []config.VertexCompatKey{
				{Name: "acct-a", APIKey: "key-a", BaseURL: "https://a.example.com"},
			},
		},
		configFilePath: writeTestConfigFile(t),
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodDelete, "/v0/management/vertex-api-key?name=missing", nil)

	h.DeleteVertexCompatKey(c)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if got := len(h.cfg.VertexCompatAPIKey); got != 1 {
		t.Fatalf("vertex keys len = %d, want 1 (unchanged)", got)
	}
}
