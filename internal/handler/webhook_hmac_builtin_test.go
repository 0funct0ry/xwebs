package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
)

func TestWebhookHMACBuiltin(t *testing.T) {
	tests := []struct {
		name           string
		action         Action
		msg            *ws.Message
		secret         string
		expectedBody   string
		expectedStatus int
		wantErr        bool
	}{
		{
			name: "basic hmac-sha256",
			action: Action{
				URL:    "", // set in test
				Secret: "super-secret",
			},
			msg: &ws.Message{
				Data: []byte("hello world"),
			},
			secret:       "super-secret",
			expectedBody: "hello world",
		},
		{
			name: "templated secret",
			action: Action{
				URL:    "", // set in test
				Secret: "{{.Session.secret}}",
			},
			msg: &ws.Message{
				Data: []byte("hello world"),
			},
			secret:       "session-secret",
			expectedBody: "hello world",
		},
		{
			name: "templated body",
			action: Action{
				URL:    "", // set in test
				Secret: "secret",
				Body:   "prefix: {{.Message}}",
			},
			msg: &ws.Message{
				Data: []byte("hello"),
			},
			secret:       "secret",
			expectedBody: "prefix: hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				assert.Equal(t, tt.expectedBody, string(body))

				// Verify HMAC
				mac := hmac.New(sha256.New, []byte(tt.secret))
				mac.Write(body)
				expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

				assert.Equal(t, expectedSig, r.Header.Get("X-Hub-Signature-256"))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			}))
			defer server.Close()

			tt.action.URL = server.URL

			d := &Dispatcher{
				templateEngine: template.New(false),
				conn:           &mockConn{},
			}

			tmplCtx := template.NewContext()
			if tt.name == "templated secret" {
				tmplCtx.Session = map[string]interface{}{"secret": "session-secret"}
			}
			d.populateTemplateContext(tmplCtx, tt.msg)

			builtin := &WebhookHMACBuiltin{}
			err := builtin.Execute(context.Background(), d, &tt.action, tmplCtx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, tmplCtx.HttpStatus)
				assert.Equal(t, "ok", tmplCtx.HttpBody)
			}
		})
	}
}

func TestWebhookHMACBuiltin_Validation(t *testing.T) {
	builtin := &WebhookHMACBuiltin{}

	// Missing URL
	err := builtin.Validate(Action{Secret: "foo"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing url")

	// Missing Secret
	err = builtin.Validate(Action{URL: "http://example.com"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing secret")

	// Valid
	err = builtin.Validate(Action{URL: "http://example.com", Secret: "foo"})
	assert.NoError(t, err)
}
