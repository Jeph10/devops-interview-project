package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := getEnv("PORT", "8080")
	store := NewMemoryStore()

	mux := http.NewServeMux()

	// Health check — must respond 200 for liveness probes
	mux.HandleFunc("GET /healthz", HealthHandler)

	// Metrics — Prometheus-compatible plaintext endpoint
	mux.HandleFunc("GET /metrics", MetricsHandler(store))

	// Task CRUD
	mux.HandleFunc("GET /tasks", ListTasksHandler(store))
	mux.HandleFunc("POST /tasks", CreateTaskHandler(store))
	mux.HandleFunc("GET /tasks/{id}", GetTaskHandler(store))
	mux.HandleFunc("PUT /tasks/{id}", UpdateTaskHandler(store))
	mux.HandleFunc("DELETE /tasks/{id}", DeleteTaskHandler(store))

	addr := ":" + port
	log.Printf("task-api starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
