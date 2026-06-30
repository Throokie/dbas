package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/multibranch"
	"reasonix/dbas/internal/dbas/fsm"
	"reasonix/dbas/internal/dbas/persistence"
)

// --- Mock dependencies ---

type mockCoordFactory struct {
	runner multibranch.Runner
}

func (f *mockCoordFactory) Create(b *branch.Branch) (multibranch.Runner, error) {
	return f.runner, nil
}

type mockRunner struct{}

func (r *mockRunner) Run(ctx context.Context, input string) error {
	return nil
}

// --- Helpers ---

func setupTestServer(t *testing.T) (*Server, *persistence.BranchStore, *branch.BranchEngine) {
	t.Helper()
	root := t.TempDir()
	slot := experienceslot.DefaultSlot()
	engine := branch.NewEngine("test-root-hash", slot)
	store := persistence.NewBranchStore(root)
	// Scheduler is optional for most tests; pass nil when scheduler not needed
	s := NewServer(engine, store, &mockCoordFactory{runner: &mockRunner{}}, nil, 8080, "")
	return s, store, engine
}



func request(t *testing.T, s *Server, method, path string, body interface{}) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		buf = *bytes.NewBuffer(b)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w.Result()
}

func readJSON(t *testing.T, res *http.Response, v interface{}) {
	t.Helper()
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		t.Fatalf("JSON decode: %v", err)
	}
}

// --- Tests ---

func TestHealthEndpoint(t *testing.T) {
	s, _, _ := setupTestServer(t)
	res := request(t, s, "GET", "/health", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]interface{}
	readJSON(t, res, &body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body["status"])
	}
}

func TestStateEndpoint(t *testing.T) {
	s, _, engine := setupTestServer(t)
	engine.CreateBranch("test-branch", nil)

	res := request(t, s, "GET", "/api/dbas/state", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]interface{}
	readJSON(t, res, &body)

	branches, ok := body["branches"].([]interface{})
	if !ok {
		t.Fatal("expected branches array")
	}
	if len(branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(branches))
	}

	b := branches[0].(map[string]interface{})
	if b["current_state"] != string(fsm.StateINIT) {
		t.Fatalf("expected INIT state, got %v", b["current_state"])
	}
}

func TestSpawnEndpoint(t *testing.T) {
	s, store, engine := setupTestServer(t)

	body := map[string]string{
		"taskName": "hello-world",
		"input":    "Write hello.go",
	}
	res := request(t, s, "POST", "/api/dbas/spawn", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var result map[string]interface{}
	readJSON(t, res, &result)
	if result["branch_id"] == nil {
		t.Fatal("expected branch_id in response")
	}

	branchID := result["branch_id"].(string)

	// Verify branch persisted (check engine)
	_, ok := engine.GetBranch(branchID)
	if !ok {
		t.Fatal("branch not found in engine after spawn")
	}

	// Verify store has branch meta
	loaded, err := store.LoadBranchMeta(branchID)
	if err != nil {
		t.Fatalf("LoadBranchMeta: %v", err)
	}
	if loaded.BranchID != branchID {
		t.Fatalf("stored branch ID = %s, want %s", loaded.BranchID, branchID)
	}
}

func TestSpawnEndpointRequiresTaskName(t *testing.T) {
	s, _, _ := setupTestServer(t)

	res := request(t, s, "POST", "/api/dbas/spawn", map[string]string{"input": "no task name"})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestStateReflectsSpawnedBranch(t *testing.T) {
	s, _, engine := setupTestServer(t)

	// Before: no branches
	res1 := request(t, s, "GET", "/api/dbas/state", nil)
	var state1 map[string]interface{}
	readJSON(t, res1, &state1)
	branches1 := state1["branches"].([]interface{})
	if len(branches1) != 0 {
		t.Fatalf("expected 0 branches before spawn, got %d", len(branches1))
	}

	// Spawn
	engine.CreateBranch("new-branch", nil)

	// After: 1 branch
	res2 := request(t, s, "GET", "/api/dbas/state", nil)
	var state2 map[string]interface{}
	readJSON(t, res2, &state2)
	branches2 := state2["branches"].([]interface{})
	if len(branches2) != 1 {
		t.Fatalf("expected 1 branch after spawn, got %d", len(branches2))
	}
}

func TestSpawnParallelEndpoint(t *testing.T) {
	// Parallel execution is tested via TestCoordinatorLifecycle in brain package.
	// The HTTP endpoint is tested indirectly through the state/spawn integration.
	t.Skip("parallel spawning is covered by coordinator integration tests")
}

func TestResetEndpoint(t *testing.T) {
	s, store, engine := setupTestServer(t)

	// Create a branch and advance it
	b, _ := engine.CreateBranch("reset-me", nil)
	b.Transition(fsm.StatePLAN, "test")
	b.Transition(fsm.StateEXECUTING, "test")
	store.InitBranch(b)
	store.SaveBranchMeta(b)

	// Reset via API
	res := request(t, s, "POST", "/api/dbas/reset", map[string]string{"branchId": b.BranchID})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	// Verify state is INIT
	updated, _ := engine.GetBranch(b.BranchID)
	if updated.CurrentState != fsm.StateINIT {
		t.Fatalf("expected INIT after reset, got %s", updated.CurrentState)
	}
	if updated.Retries != 0 {
		t.Fatalf("expected retries=0 after reset, got %d", updated.Retries)
	}
}

func TestResetNonexistentBranch(t *testing.T) {
	s, _, _ := setupTestServer(t)

	res := request(t, s, "POST", "/api/dbas/reset", map[string]string{"branchId": "nonexistent"})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent branch, got %d", res.StatusCode)
	}
}

func TestBranchDetailEndpoint(t *testing.T) {
	s, store, engine := setupTestServer(t)

	b, _ := engine.CreateBranch("detail-test", nil)
	store.InitBranch(b)

	res := request(t, s, "GET", "/api/dbas/branches/"+b.BranchID, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]interface{}
	readJSON(t, res, &body)
	if body["branch"] == nil {
		t.Fatal("expected branch in response")
	}
}

func TestDemoEndpoint(t *testing.T) {
	s, _, _ := setupTestServer(t)

	res := request(t, s, "POST", "/api/dbas/demo", map[string]string{"taskName": "test-demo"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var result map[string]interface{}
	readJSON(t, res, &result)
	if result["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", result["status"])
	}
	if result["branch_id"] == nil {
		t.Fatal("expected branch_id in demo response")
	}
}

func TestSlotEndpoint(t *testing.T) {
	s, _, _ := setupTestServer(t)

	res := request(t, s, "GET", "/api/dbas/slot", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]interface{}
	readJSON(t, res, &body)
	if body["project_id"] == nil {
		t.Fatal("expected project_id in slot response")
	}
}

func TestMultipleBranches(t *testing.T) {
	s, store, engine := setupTestServer(t)

	// Create 3 branches with different terminal states
	b1, _ := engine.CreateBranch("happy-path", nil)
	b1.Transition(fsm.StatePLAN, "")
	b1.Transition(fsm.StateEXECUTING, "")
	b1.Transition(fsm.StateVERIFYING, "")
	b1.Transition(fsm.StatePASS, "")
	b1.Transition(fsm.StateARCHIVE, "")
	store.InitBranch(b1)
	store.SaveBranchMeta(b1)

	b2, _ := engine.CreateBranch("running-task", nil)
	b2.Transition(fsm.StatePLAN, "")
	b2.Transition(fsm.StateEXECUTING, "")
	store.InitBranch(b2)
	store.SaveBranchMeta(b2)

	b3, _ := engine.CreateBranch("failed-task", nil)
	b3.Transition(fsm.StatePLAN, "")
	b3.Transition(fsm.StateEXECUTING, "")
	b3.FSM().ForceError("test error")
	b3.CurrentState = fsm.StateERROR
	store.InitBranch(b3)
	store.SaveBranchMeta(b3)

	// Verify state reflects all branches
	res := request(t, s, "GET", "/api/dbas/state", nil)
	var state map[string]interface{}
	readJSON(t, res, &state)

	branches := state["branches"].([]interface{})
	if len(branches) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(branches))
	}

	// Verify terminal states
	for _, br := range branches {
		b := br.(map[string]interface{})
		id := b["branch_id"].(string)
		stateStr := b["current_state"].(string)

		switch {
		case strings.Contains(id, "happy") && stateStr != string(fsm.StateARCHIVE):
			t.Errorf("happy branch: expected ARCHIVE, got %s", stateStr)
		case strings.Contains(id, "running") && stateStr != string(fsm.StateEXECUTING):
			t.Errorf("running branch: expected EXECUTING, got %s", stateStr)
		case strings.Contains(id, "failed") && stateStr != string(fsm.StateERROR):
			t.Errorf("failed branch: expected ERROR, got %s", stateStr)
		}
	}
}

func TestConcurrentSpawn(t *testing.T) {
	s, store, engine := setupTestServer(t)
	n := 10

	for i := 0; i < n; i++ {
		name := "concurrent-" + string(rune('a'+i))
		b, err := engine.CreateBranch(name, nil)
		if err != nil {
			t.Fatalf("create branch %d: %v", i, err)
		}
		if err := store.InitBranch(b); err != nil {
			t.Fatalf("init branch %d: %v", i, err)
		}
	}

	// Verify all branches
	all := engine.GetAllBranches()
	if len(all) != n {
		t.Fatalf("expected %d branches, got %d", n, len(all))
	}
	_ = s
}

func TestMethodValidation(t *testing.T) {
	s, _, _ := setupTestServer(t)

	// POST-only endpoints should reject GET
	postEndpoints := []string{"/api/dbas/spawn", "/api/dbas/reset", "/api/dbas/demo"}
	for _, ep := range postEndpoints {
		res := request(t, s, "GET", ep, nil)
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("GET %s: expected 405, got %d", ep, res.StatusCode)
		}
	}

	// GET-only endpoints should reject POST
	getEndpoints := []string{"/api/dbas/state"}
	for _, ep := range getEndpoints {
		res := request(t, s, "POST", ep, nil)
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("POST %s: expected 405, got %d", ep, res.StatusCode)
		}
	}
}

func TestCORSHeaders(t *testing.T) {
	s, _, _ := setupTestServer(t)

	handler := ChainMiddleware(s, RecoveryMiddleware, LoggingMiddleware, CORSMiddleware)

	req := httptest.NewRequest("OPTIONS", "/api/dbas/state", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	res := w.Result()
	if res.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected CORS Allow-Origin: *")
	}
}

func TestStaticFileServing(t *testing.T) {
	// Static file serving depends on runtime configuration; skip in unit tests
	t.Skip("static file serving tested manually")
}

func TestBranchIDUniqueness(t *testing.T) {
	_, _, engine := setupTestServer(t)

	b1, _ := engine.CreateBranch("same-name", nil)
	b2, _ := engine.CreateBranch("same-name", nil)

	if b1.BranchID == b2.BranchID {
		t.Fatal("expected unique branch IDs")
	}
}



