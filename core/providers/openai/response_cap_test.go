package openai

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
)

func TestChatCompletionHardResponseBodyLimit(t *testing.T) {
	t.Parallel()

	const maximum = 512
	validBody := []byte(`{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
	largeBody := append([]byte(`{"padding":"`), bytes.Repeat([]byte("x"), maximum)...)
	largeBody = append(largeBody, []byte(`"}`)...)
	gzipBomb := gzipEncode(t, append([]byte(`{"padding":"`), append(bytes.Repeat([]byte("x"), maximum*4), []byte(`"}`)...)...))
	if len(gzipBomb) >= maximum {
		t.Fatalf("test gzip body is not compressed below cap: %d", len(gzipBomb))
	}

	tests := []struct {
		name       string
		status     int
		body       []byte
		gzip       bool
		chunked    bool
		wantError  bool
		errorMatch string
	}{
		{name: "success within cap", status: http.StatusOK, body: validBody},
		{name: "content length success over cap", status: http.StatusOK, body: largeBody, wantError: true, errorMatch: "body size exceeds"},
		{name: "chunked success over cap", status: http.StatusOK, body: largeBody, chunked: true, wantError: true, errorMatch: "body size exceeds"},
		{name: "content length error over cap", status: http.StatusBadGateway, body: largeBody, wantError: true, errorMatch: "body size exceeds"},
		{name: "gzip decoded bomb", status: http.StatusOK, body: gzipBomb, gzip: true, wantError: true, errorMatch: "exceeds configured limit"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if test.gzip {
					w.Header().Set("Content-Encoding", "gzip")
				}
				w.WriteHeader(test.status)
				if test.chunked {
					flusher, ok := w.(http.Flusher)
					if !ok {
						t.Error("response writer does not support flushing")
						return
					}
					for offset := 0; offset < len(test.body); offset += 64 {
						end := min(offset+64, len(test.body))
						_, _ = w.Write(test.body[offset:end])
						flusher.Flush()
					}
					return
				}
				_, _ = w.Write(test.body)
			}))
			defer server.Close()

			provider := NewOpenAIProvider(&schemas.ProviderConfig{
				NetworkConfig: schemas.NetworkConfig{
					BaseURL:              server.URL,
					AllowPrivateNetwork:  true,
					MaxResponseBodyBytes: maximum,
				},
			}, testNoopLogger{})
			ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
			response, bifrostErr := provider.ChatCompletion(ctx, testKey(), basicChatRequest())
			if test.wantError {
				if bifrostErr == nil {
					t.Fatalf("expected response-size error, got response %+v", response)
				}
				message := ""
				if bifrostErr.Error != nil {
					message = bifrostErr.Error.Message
					if bifrostErr.Error.Error != nil {
						message += " " + bifrostErr.Error.Error.Error()
					}
				}
				if !strings.Contains(strings.ToLower(message), test.errorMatch) {
					t.Fatalf("error = %+v, want message containing %q", bifrostErr, test.errorMatch)
				}
				return
			}
			if bifrostErr != nil {
				t.Fatalf("chat completion: %+v", bifrostErr)
			}
			if response == nil || len(response.Choices) != 1 {
				t.Fatalf("response = %+v, want one choice", response)
			}
		})
	}
}

func gzipEncode(t *testing.T, body []byte) []byte {
	t.Helper()
	var encoded bytes.Buffer
	writer := gzip.NewWriter(&encoded)
	if _, err := writer.Write(body); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return encoded.Bytes()
}
