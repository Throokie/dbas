package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/fsm"
	"reasonix/dbas/internal/dbas/multibranch"
)

type Server struct {
	mux                *http.ServeMux
	engine             *branch.BranchEngine
	store              branchStore
	coordinatorFactory CoordinatorFactory
	scheduler          *multibranch.Scheduler
	frontendDir        string
	port               int
}

type branchStore interface {
	InitBranch(b *branch.Branch) error
	SaveBranchMeta(b *branch.Branch) error
	AppendLog(branchID string, entry string) error
	ListBranches() ([]string, error)
}

type CoordinatorFactory interface {
	Create(b *branch.Branch) (multibranch.Runner, error)
}

type Runner interface {
	Run(ctx context.Context, input string) error
}

func NewServer(engine *branch.BranchEngine, store branchStore, factory CoordinatorFactory, scheduler *multibranch.Scheduler, port int, frontendDir string) *Server {
	if port <= 0 {
		port = 8080
	}
	s := &Server{
		mux:                http.NewServeMux(),
		engine:             engine,
		store:              store,
		coordinatorFactory: factory,
		scheduler:          scheduler,
		frontendDir:        frontendDir,
		port:               port,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/dbas/state", s.handleState)
	s.mux.HandleFunc("/api/dbas/spawn", s.handleSpawn)
	s.mux.HandleFunc("/api/dbas/spawn-parallel", s.handleSpawnParallel)
	s.mux.HandleFunc("/api/dbas/reset", s.handleReset)
	s.mux.HandleFunc("/api/dbas/branches/", s.handleBranchDetail)
	s.mux.HandleFunc("/api/dbas/demo", s.handleDemo)
	s.mux.HandleFunc("/api/dbas/slot", s.handleSlot)
	s.mux.HandleFunc("/health", s.handleHealth)

	if s.frontendDir != "" {
		s.mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
				http.NotFound(w, r)
				return
			}
			target := filepath.Join(s.frontendDir, "index.html")
			path := filepath.Join(s.frontendDir, filepath.Clean(r.URL.Path))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				target = path
			}
			data, err := os.ReadFile(target)
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			ct := detectContentType(target)
			if ct == "" {
				ct = "text/plain; charset=utf-8"
			}
			w.Header().Set("Content-Type", ct)
			w.Write(data)
		}))
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.port)
	slog.Info("DBAS API server starting", "addr", addr)
	return http.ListenAndServe(addr, s)
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	branches := s.engine.GetAllBranches()
	response := map[string]interface{}{
		"experience_slot": s.engine.ExperienceSlot(),
		"branches":        branches,
		"timestamp":       time.Now().Unix(),
	}
	writeJSON(w, response)
}

func (s *Server) handleSpawn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TaskName string  `json:"taskName"`
		ParentID *string `json:"parentId,omitempty"`
		Input    string  `json:"input,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.TaskName == "" {
		http.Error(w, "taskName is required", http.StatusBadRequest)
		return
	}
	b, err := s.engine.CreateBranch(req.TaskName, req.ParentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.store.InitBranch(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := map[string]interface{}{
		"status":    "ok",
		"branch_id": b.BranchID,
	}
	if s.scheduler != nil {
		inputs := []multibranch.TaskInput{
			{Name: req.TaskName, Input: req.Input},
		}
		if _, err := s.scheduler.SpawnSequential(context.Background(), "", inputs); err != nil {
			slog.Warn("scheduler spawn", "err", err)
		}
		response["scheduled"] = true
	}
	writeJSON(w, response)
}

func (s *Server) handleSpawnParallel(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status":  "ok",
		"message": "Parallel spawning is covered by coordinator integration tests.",
	})
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		BranchID string `json:"branchId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.BranchID == "" {
		http.Error(w, "branchId is required", http.StatusBadRequest)
		return
	}
	if err := s.engine.ResetBranch(req.BranchID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	b, _ := s.engine.GetBranch(req.BranchID)
	writeJSON(w, b)
}

func (s *Server) handleBranchDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Path[len("/api/dbas/branches/"):]
	if id == "" {
		http.Error(w, "branch ID required", http.StatusBadRequest)
		return
	}
	b, ok := s.engine.GetBranch(id)
	if !ok {
		http.Error(w, "Branch not found", http.StatusNotFound)
		return
	}
	response := map[string]interface{}{
		"branch": b,
	}
	if b.OutputPaths.ResultReport != "" {
		data, err := os.ReadFile(b.OutputPaths.ResultReport)
		if err == nil {
			var report map[string]interface{}
			if json.Unmarshal(data, &report) == nil {
				response["report"] = report
			}
		}
	}
	if b.OutputPaths.LogFile != "" {
		data, err := os.ReadFile(b.OutputPaths.LogFile)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			if len(lines) > 50 {
				lines = lines[len(lines)-50:]
			}
			var nonEmpty []string
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					nonEmpty = append(nonEmpty, line)
				}
			}
			response["log_tail"] = nonEmpty
		}
	}
	writeJSON(w, response)
}

func (s *Server) handleDemo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TaskName string `json:"taskName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.TaskName = "demo"
	}
	if req.TaskName == "" {
		req.TaskName = "demo"
	}
	b, err := s.engine.CreateBranch(req.TaskName+"-demo", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.store.InitBranch(b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go func(b *branch.Branch) {
		steps := []struct {
			state  fsm.State
			reason string
			delay  time.Duration
		}{
			{fsm.StatePLAN, "Decider: task analysis complete", 500 * time.Millisecond},
			{fsm.StateEXECUTING, "Reviewer: plan approved", 800 * time.Millisecond},
			{fsm.StateVERIFYING, "Executor: code changes complete", 600 * time.Millisecond},
			{fsm.StatePASS, "Verifier: all checks passed", 400 * time.Millisecond},
			{fsm.StateARCHIVE, "Cross-auditor: evidence verified", 300 * time.Millisecond},
		}
		b.Logs = append(b.Logs, "[DEMO] Starting FSM lifecycle demo...")
		for _, step := range steps {
			time.Sleep(step.delay)
			if err := b.Transition(step.state, step.reason); err != nil {
				b.Logs = append(b.Logs, fmt.Sprintf("[DEMO_ERROR] %v", err))
				break
			}
			if err := s.store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1]); err != nil {
				slog.Error("demo append log", "err", err)
			}
			if err := s.store.SaveBranchMeta(b); err != nil {
				slog.Error("demo save meta", "err", err)
			}
		}
		b.Logs = append(b.Logs, "[DEMO] Lifecycle complete - check the dashboard!")
		_ = s.store.AppendLog(b.BranchID, b.Logs[len(b.Logs)-1])
		_ = s.store.SaveBranchMeta(b)
	}(b)
	writeJSON(w, map[string]interface{}{
		"status":    "ok",
		"branch_id": b.BranchID,
		"message":   "Demo branch spawned. Watch the lifecycle unfold on the dashboard.",
	})
}

func (s *Server) handleSlot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.engine.ExperienceSlot())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
	w.Write([]byte{'\n'})
}



func detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	default:
		return mime.TypeByExtension(ext)
	}
}



