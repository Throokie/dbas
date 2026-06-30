package api

import (
	"net/http"

	"reasonix/dbas/internal/dbas/plugin"
	"reasonix/dbas/internal/dbas/skill"
)

// ExtHandler provides HTTP endpoints for extensions discovery.
type ExtHandler struct {
	skillLoader *skill.Loader
	registry    *plugin.Registry
}

// NewExtHandler creates a handler for extension-related API endpoints.
func NewExtHandler(sl *skill.Loader, reg *plugin.Registry) *ExtHandler {
	return &ExtHandler{skillLoader: sl, registry: reg}
}

// RegisterRoutes adds extension endpoints to the server.
func (h *ExtHandler) RegisterRoutes(s *Server) {
	s.mux.HandleFunc("/api/dbas/skills", h.handleSkills)
	s.mux.HandleFunc("/api/dbas/plugins", h.handlePlugins)
	s.mux.HandleFunc("/api/dbas/extensions", h.handleExtensions)
}

func (h *ExtHandler) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	skills, err := h.skillLoader.LoadAll()
	if err != nil {
		writeJSON(w, map[string]interface{}{"count": 0, "skills": []interface{}{}, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"count":  len(skills),
		"skills": skills,
	})
}

func (h *ExtHandler) handlePlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]interface{}{
		"executors": h.registry.ListExecutors(),
		"verifiers": h.registry.ListVerifiers(),
		"notifiers": h.registry.ListNotifiers(),
	})
}

func (h *ExtHandler) handleExtensions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	extensions := h.registry.ListExtensions()
	writeJSON(w, map[string]interface{}{
		"count":      len(extensions),
		"extensions": extensions,
	})
}
