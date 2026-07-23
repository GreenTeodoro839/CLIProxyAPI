package claude

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/interfaces"
	"github.com/tidwall/gjson"
)

func TestClaudeErrorExtractsOpenAIStyleUpstreamJSON(t *testing.T) {
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadRequest,
		Error:      errors.New(`{"error":{"message":"Your input exceeds the context window of this model. Please adjust your input and try again.","type":"invalid_request_error","code":"context_too_large"}}`),
	}

	got := handler.toClaudeError(msg)

	if got.Type != "error" {
		t.Fatalf("type = %q, want error", got.Type)
	}
	if got.Error.Type != "invalid_request_error" {
		t.Fatalf("error.type = %q, want invalid_request_error", got.Error.Type)
	}
	if got.Error.Message != "prompt is too long: Your input exceeds the context window of this model. Please adjust your input and try again." {
		t.Fatalf("error.message = %q", got.Error.Message)
	}
}

func TestClaudeErrorNormalizesContextLimitSignalsForClaudeCode(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		errText     string
		wantType    string
		wantMessage string
	}{
		{
			name:        "top-level Codex context code",
			status:      http.StatusBadRequest,
			errText:     `{"code":"context_length_exceeded","message":"Input rejected."}`,
			wantType:    "invalid_request_error",
			wantMessage: "prompt is too long: Input rejected.",
		},
		{
			name:        "native compaction hard boundary",
			status:      http.StatusBadRequest,
			errText:     "codex native compaction failed at the configured 272000-token context boundary: upstream rejected compaction",
			wantType:    "invalid_request_error",
			wantMessage: "prompt is too long: codex native compaction failed at the configured 272000-token context boundary: upstream rejected compaction",
		},
		{
			name:        "already compatible",
			status:      http.StatusBadRequest,
			errText:     "prompt is too long: 280000 tokens > 272000",
			wantType:    "invalid_request_error",
			wantMessage: "prompt is too long: 280000 tokens > 272000",
		},
		{
			name:        "unrelated invalid request",
			status:      http.StatusBadRequest,
			errText:     "invalid tool schema",
			wantType:    "invalid_request_error",
			wantMessage: "invalid tool schema",
		},
		{
			name:        "large request body is not a context error",
			status:      http.StatusRequestEntityTooLarge,
			errText:     "request body exceeds the maximum allowed size",
			wantType:    "request_too_large",
			wantMessage: "request body exceeds the maximum allowed size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotMessage := claudeErrorDetailFromText(tt.status, tt.errText)
			if gotType != tt.wantType {
				t.Fatalf("error.type = %q, want %q", gotType, tt.wantType)
			}
			if gotMessage != tt.wantMessage {
				t.Fatalf("error.message = %q, want %q", gotMessage, tt.wantMessage)
			}
		})
	}
}

func TestClaudeErrorExtractsClaudeStyleUpstreamJSON(t *testing.T) {
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusTooManyRequests,
		Error:      errors.New(`{"type":"error","error":{"type":"rate_limit_error","message":"This request would exceed your account's rate limit. Please try again later."},"request_id":"req_123"}`),
	}

	got := handler.toClaudeError(msg)

	if got.Error.Type != "rate_limit_error" {
		t.Fatalf("error.type = %q, want rate_limit_error", got.Error.Type)
	}
	if got.Error.Message != "This request would exceed your account's rate limit. Please try again later." {
		t.Fatalf("error.message = %q", got.Error.Message)
	}
}

func TestWriteClaudeErrorResponseUsesClaudeEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	handler := &ClaudeCodeAPIHandler{}
	msg := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadRequest,
		Error:      errors.New(`{"error":{"message":"Your input exceeds the context window of this model. Please adjust your input and try again.","type":"invalid_request_error","code":"context_too_large"}}`),
	}

	handler.WriteErrorResponse(c, msg)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	body := recorder.Body.Bytes()
	if got := gjson.GetBytes(body, "type").String(); got != "error" {
		t.Fatalf("type = %q, want error; body=%s", got, body)
	}
	if got := gjson.GetBytes(body, "error.type").String(); got != "invalid_request_error" {
		t.Fatalf("error.type = %q, want invalid_request_error; body=%s", got, body)
	}
	if got := gjson.GetBytes(body, "error.message").String(); got != "prompt is too long: Your input exceeds the context window of this model. Please adjust your input and try again." {
		t.Fatalf("error.message = %q; body=%s", got, body)
	}
}

func TestPendingClaudeStreamErrorUsesBufferedError(t *testing.T) {
	wantErr := &interfaces.ErrorMessage{
		StatusCode: http.StatusBadRequest,
		Error:      errors.New(`{"error":{"message":"Your input exceeds the context window of this model. Please adjust your input and try again.","type":"invalid_request_error","code":"context_too_large"}}`),
	}
	errs := make(chan *interfaces.ErrorMessage, 1)
	errs <- wantErr
	close(errs)

	gotErr, ok := pendingClaudeStreamError(errs)
	if !ok {
		t.Fatal("expected pending stream error")
	}
	if gotErr != wantErr {
		t.Fatalf("pending error = %p, want %p", gotErr, wantErr)
	}
}
