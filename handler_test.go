package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func setup() *MemoryStore {
	return NewMemoryStore()
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
}

func TestCreateAndGetTask(t *testing.T) {
	store := setup()

	// Create
	payload := `{"title": "ship it"}`
	req := httptest.NewRequest("POST", "/tasks", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	CreateTaskHandler(store)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created Task
	json.NewDecoder(rec.Body).Decode(&created)
	if created.Title != "ship it" {
		t.Fatalf("expected title 'ship it', got %q", created.Title)
	}

	// Get
	req = httptest.NewRequest("GET", "/tasks/0", nil)
	req.SetPathValue("id", "0")
	rec = httptest.NewRecorder()
	GetTaskHandler(store)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var fetched Task
	json.NewDecoder(rec.Body).Decode(&fetched)
	if fetched.ID != 0 || fetched.Title != "ship it" {
		t.Fatalf("unexpected task: %+v", fetched)
	}
}

func TestUpdateTask(t *testing.T) {
	store := setup()
	store.Create("learn k8s")

	done := true
	body, _ := json.Marshal(map[string]any{"done": done})
	req := httptest.NewRequest("PUT", "/tasks/0", bytes.NewReader(body))
	req.SetPathValue("id", "0")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	UpdateTaskHandler(store)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	task, _ := store.Get(0)
	if !task.Done {
		t.Fatalf("expected task to be done")
	}
}

func TestDeleteTask(t *testing.T) {
	store := setup()
	store.Create("delete me")

	req := httptest.NewRequest("DELETE", "/tasks/0", nil)
	req.SetPathValue("id", "0")
	rec := httptest.NewRecorder()
	DeleteTaskHandler(store)(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	if _, ok := store.Get(0); ok {
		t.Fatalf("task should have been deleted")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	store := setup()
	store.Create("task A")
	task, _ := store.Get(0)
	done := true
	store.Update(task.ID, nil, &done)
	store.Create("task B")

	// Create middleware with a fresh registry to avoid duplicate registration.
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMiddlewareWithRegistry(reg)
	pm.UpdateTaskStats(2, 1)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	pm.MetricsHandlerForRegistry(reg).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "task_api_tasks_total 2") {
		t.Fatalf("expected total=2 in metrics, got:\n%s", body)
	}
	if !strings.Contains(body, "task_api_tasks_done 1") {
		t.Fatalf("expected done=1 in metrics, got:\n%s", body)
	}
}

func TestMetricsCounterIncremented(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMiddlewareWithRegistry(reg)

	// Simulate a request.
	pm.RecordRequest("GET", "/tasks", "200", 0.05)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	pm.MetricsHandlerForRegistry(reg).ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `http_requests_total{method="GET",path="/tasks",status="200"} 1`) {
		t.Fatalf("expected counter to be incremented, got:\n%s", body)
	}
}

func TestMetricsDurationRecorded(t *testing.T) {
	reg := prometheus.NewRegistry()
	pm := NewPrometheusMiddlewareWithRegistry(reg)

	// Simulate a request with duration.
	pm.RecordRequest("POST", "/tasks", "201", 0.1)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	pm.MetricsHandlerForRegistry(reg).ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `http_request_duration_seconds_count{method="POST",path="/tasks"} 1`) {
		t.Fatalf("expected histogram to have observation, got:\n%s", body)
	}
}

func TestCreateTaskMissingTitle(t *testing.T) {
	store := setup()
	req := httptest.NewRequest("POST", "/tasks", strings.NewReader(`{"title":""}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	CreateTaskHandler(store)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	store := setup()
	req := httptest.NewRequest("GET", "/tasks/999", nil)
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()
	GetTaskHandler(store)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
