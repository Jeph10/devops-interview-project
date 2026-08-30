package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Healthcheck subcommand: probe /healthz and exit 0/1.
	// Lets the Docker HEALTHCHECK work on distroless images that have
	// no shell, curl, or wget.
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz and exit with its status")
	flag.Parse()
	if *healthcheck {
		runHealthcheck()
		return
	}

	port := getEnv("PORT", "8080")
	store := NewMemoryStore()

	// Create Prometheus metrics middleware.
	pm := NewPrometheusMiddleware()

	mux := http.NewServeMux()

	// Health check — must respond 200 for liveness probes
	mux.HandleFunc("GET /healthz", HealthHandler)

	// Metrics — Prometheus /metrics endpoint (not wrapped in RED metrics)
	mux.Handle("GET /metrics", pm.MetricsHandler())

	// Task CRUD — wrapped with Prometheus RED metrics
	mux.HandleFunc("GET /tasks", pm.WrapHandler("GET /tasks", ListTasksHandler(store)))
	mux.HandleFunc("POST /tasks", pm.WrapHandler("POST /tasks", CreateTaskHandler(store)))
	mux.HandleFunc("GET /tasks/{id}", pm.WrapHandler("GET /tasks/{id}", GetTaskHandler(store)))
	mux.HandleFunc("PUT /tasks/{id}", pm.WrapHandler("PUT /tasks/{id}", UpdateTaskHandler(store)))
	mux.HandleFunc("DELETE /tasks/{id}", pm.WrapHandler("DELETE /tasks/{id}", DeleteTaskHandler(store)))

	// Periodically update task gauge metrics.
	go func() {
		for {
			total, done := store.Stats()
			pm.UpdateTaskStats(total, done)
			time.Sleep(5 * time.Second)
		}
	}()

	addr := ":" + port
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown on SIGTERM/SIGINT so the container stops cleanly.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig
		log.Println("task-api: shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	log.Printf("task-api starting on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

// runHealthcheck probes the local /healthz endpoint and exits with code 0 on
// success or code 1 on failure. Used by the Docker HEALTHCHECK instruction.
func runHealthcheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8080/healthz", nil)
	if err != nil {
		log.Fatalf("healthcheck request build failed: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("healthcheck failed: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("healthcheck: unexpected status %d", resp.StatusCode)
		os.Exit(1)
	}
	os.Exit(0)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
