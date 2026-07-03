package synthesizer

import (
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

// TestConfigSynthesizer_NameAttribute asserts that an optional Name on each
// non-OpenAI provider config is carried onto the runtime auth under the "name"
// attribute (used by the api-key-usage endpoint), without setting compat_name
// (which would misclassify the auth as OpenAI-compatible).
func TestConfigSynthesizer_NameAttribute(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		GeminiKey: []config.GeminiKey{
			{APIKey: "gem-key", Name: "gem-acct"},
		},
		ClaudeKey: []config.ClaudeKey{
			{APIKey: "claude-key", Name: "claude-acct"},
		},
		CodexKey: []config.CodexKey{
			{APIKey: "codex-key", BaseURL: "https://codex.example.com", Name: "codex-acct"},
		},
		VertexCompatAPIKey: []config.VertexCompatKey{
			{APIKey: "vertex-key", Name: "vertex-acct"},
		},
	}
	synth := NewConfigSynthesizer()
	ctx := &SynthesisContext{
		Config:      cfg,
		Now:         time.Now(),
		IDGenerator: NewStableIDGenerator(),
	}
	auths, err := synth.Synthesize(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]string{
		"gemini": "gem-acct",
		"claude": "claude-acct",
		"codex":  "codex-acct",
		"vertex": "vertex-acct",
	}
	seen := map[string]string{}
	for _, a := range auths {
		if a == nil || a.Attributes == nil {
			continue
		}
		if name, ok := a.Attributes["name"]; ok {
			seen[a.Provider] = name
		}
	}
	for provider, wantName := range want {
		gotName, ok := seen[provider]
		if !ok {
			t.Errorf("%s: missing name attribute on synthesized auth", provider)
			continue
		}
		if gotName != wantName {
			t.Errorf("%s: name attribute = %q, want %q", provider, gotName, wantName)
		}
	}

	// Empty Name must not set the "name" attribute and must never set compat_name
	// (a non-OpenAI auth carrying compat_name would be misclassified).
	cfg2 := &config.Config{
		GeminiKey: []config.GeminiKey{{APIKey: "no-name-key"}},
	}
	ctx2 := &SynthesisContext{Config: cfg2, Now: time.Now(), IDGenerator: NewStableIDGenerator()}
	auths2, err := synth.Synthesize(ctx2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(auths2) != 1 {
		t.Fatalf("expected 1 auth, got %d", len(auths2))
	}
	attrs := auths2[0].Attributes
	if _, ok := attrs["name"]; ok {
		t.Errorf("unexpected name attribute when Name unset: %v", attrs)
	}
	if v := attrs["compat_name"]; v != "" {
		t.Errorf("gemini auth must not carry compat_name (would misclassify), got %q", v)
	}
}
