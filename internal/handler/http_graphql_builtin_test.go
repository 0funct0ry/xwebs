package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpGraphQLBuiltin(t *testing.T) {
	// Setup a mock GraphQL server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		query := payload["query"].(string)
		
		if query == "error" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"errors": [{"message": "something went wrong"}]}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"hello": "world",
				"vars":  payload["variables"],
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	engine := template.New(false)
	d := &Dispatcher{
		templateEngine: engine,
		verbose:        true,
		conn:           &ws.Connection{},
	}

	builtin := &HttpGraphQLBuiltin{}

	t.Run("successful query with variables", func(t *testing.T) {
		a := &Action{
			URL:       server.URL,
			Query:     "query { hello }",
			Variables: `{"id": "{{.Message}}"}`,
		}
		tmplCtx := template.NewContext()
		tmplCtx.Message = "123"

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, tmplCtx.HttpStatus)
		assert.Contains(t, tmplCtx.HttpBody, `"hello":"world"`)
		assert.Contains(t, tmplCtx.HttpBody, `"vars":{"id":"123"}`)
		assert.Nil(t, tmplCtx.GraphQLErrors)
	})

	t.Run("query with errors", func(t *testing.T) {
		a := &Action{
			URL:   server.URL,
			Query: "error",
		}
		tmplCtx := template.NewContext()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, tmplCtx.HttpStatus)
		// HttpBody might be empty or "{}" depending on how it's handled
		require.NotNil(t, tmplCtx.GraphQLErrors)
		errors := tmplCtx.GraphQLErrors.([]interface{})
		assert.Len(t, errors, 1)
		assert.Equal(t, "something went wrong", errors[0].(map[string]interface{})["message"])
	})

	t.Run("timeout", func(t *testing.T) {
		slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer slowServer.Close()

		a := &Action{
			URL:     slowServer.URL,
			Query:   "{}",
			Timeout: "50ms",
		}
		tmplCtx := template.NewContext()

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deadline exceeded")
	})
}
