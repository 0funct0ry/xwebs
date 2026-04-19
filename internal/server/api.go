package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/kv"
)

// jsonResponse is a helper to write JSON responses with status codes.
func (s *Server) jsonResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			if s.opts.Verbose {
				fmt.Printf("[api] error encoding json: %v\n", err)
			}
		}
	}
}

// errorResponse is a helper to write error JSON responses.
func (s *Server) errorResponse(w http.ResponseWriter, code int, message string) {
	s.jsonResponse(w, code, map[string]string{"error": message})
}

// handleAPIStatus returns the server's running status and basic stats.
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	connCount := len(s.connections)
	s.mu.Unlock()

	uptime := time.Since(s.startTime).Round(time.Second)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "running",
		"uptime":      uptime.String(),
		"uptime_sec":  int64(uptime.Seconds()),
		"connections": connCount,
		"ports":       []int{s.opts.Port},
		"paths":       s.opts.Paths,
		"tls":         s.opts.TLSEnabled,
	})
}

// handleAPIClients returns a list of connected clients and their metadata.
func (s *Server) handleAPIClients(w http.ResponseWriter, r *http.Request) {
	clients := s.GetClients()
	s.jsonResponse(w, http.StatusOK, clients)
}

// handleListHandlers returns the current handler registry.
func (s *Server) handleListHandlers(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, s.registry.Handlers())
}

// handleGetHandler returns a single handler by name.
func (s *Server) handleGetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	h, ok := s.registry.GetHandler(name)
	if !ok {
		s.errorResponse(w, http.StatusNotFound, fmt.Sprintf("handler %q not found", name))
		return
	}

	s.jsonResponse(w, http.StatusOK, h)
}

// handleCreateHandler adds a new handler to the registry.
func (s *Server) handleCreateHandler(w http.ResponseWriter, r *http.Request) {
	var h handler.Handler
	if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
		s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %v", err))
		return
	}

	if h.Name == "" {
		s.errorResponse(w, http.StatusBadRequest, "handler name is required")
		return
	}

	if err := s.registry.Add(h); err != nil {
		s.errorResponse(w, http.StatusConflict, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusCreated, h)
}

// handleUpdateHandler updates an existing handler.
func (s *Server) handleUpdateHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var h handler.Handler
	if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
		s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %v", err))
		return
	}

	h.Name = name // Ensure name from URL is used

	if err := s.registry.UpdateHandler(h); err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusOK, h)
}

// handleDeleteHandler removes a handler from the registry.
func (s *Server) handleDeleteHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	if err := s.registry.Delete(name); err != nil {
		s.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	s.jsonResponse(w, http.StatusNoContent, nil)
}

// handleListKV returns all entries in the key-value store.
func (s *Server) handleListKV(w http.ResponseWriter, r *http.Request) {
	if s.kvStore == nil {
		s.jsonResponse(w, http.StatusOK, make(map[string]interface{}))
		return
	}
	s.jsonResponse(w, http.StatusOK, s.kvStore.List())
}

// handleGetKV returns the value for a specific key.
func (s *Server) handleGetKV(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if s.kvStore == nil {
		s.errorResponse(w, http.StatusNotFound, "kv store not initialized")
		return
	}

	val, ok := s.kvStore.Get(key)
	if !ok {
		s.errorResponse(w, http.StatusNotFound, fmt.Sprintf("key %q not found", key))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{key: val})
}

// handleSetKV stores a value for a specific key.
func (s *Server) handleSetKV(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var val interface{}
	if err := json.NewDecoder(r.Body).Decode(&val); err != nil {
		s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %v", err))
		return
	}

	if s.kvStore == nil {
		s.kvStore = kv.NewStore()
	}

	s.kvStore.Set(key, val, 0)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{key: val})
}

// handleDeleteKV removes a key from the key-value store.
func (s *Server) handleDeleteKV(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if s.kvStore == nil {
		s.errorResponse(w, http.StatusNotFound, "kv store not initialized")
		return
	}

	s.kvStore.Delete(key)
	s.jsonResponse(w, http.StatusNoContent, nil)
}

// handleAPIHealth returns the server health status.
func (s *Server) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime).Round(time.Second)
	
	health := map[string]interface{}{
		"status":     "ok",
		"uptime":     uptime.String(),
		"uptime_sec": int64(uptime.Seconds()),
		"components": map[string]string{
			"handler_registry": "ok",
		},
	}

	code := http.StatusOK

	if s.registry == nil {
		health["status"] = "error"
		health["components"].(map[string]string)["handler_registry"] = "error"
		code = http.StatusServiceUnavailable
	}

	s.jsonResponse(w, code, health)
}

// GetClients returns a list of active connection metadata.
// Note: This is redefined here or moved from server.go if I want to keep server.go clean.
// Actually it's already in server.go, but I'll make sure it's accessible.
// Wait, ClientInfo is in template package.
