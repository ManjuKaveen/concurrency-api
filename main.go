package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
)

type Task struct {
	TaskID   string `json:"task_id"`
	Duration int    `json:"duration"`
}

type Metrics struct {
	TotalRequests         int64
	AverageProcessingTime int64
	ActiveRequests        int64
}

var (
	concurrencyLimiter chan struct{}
	metrics            Metrics
	maxConcurrency     int
)

func processHandler(w http.ResponseWriter, r *http.Request) {
	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	concurrencyLimiter <- struct{}{}
	atomic.AddInt64(&metrics.ActiveRequests, 1)

	start := time.Now()

	time.Sleep(time.Duration(task.Duration) * time.Millisecond)

	elapsed := time.Since(start).Milliseconds()
	atomic.AddInt64(&metrics.TotalRequests, 1)
	atomic.AddInt64(&metrics.AverageProcessingTime, elapsed)
	atomic.AddInt64(&metrics.ActiveRequests, -1)

	<-concurrencyLimiter

	response := map[string]string{
		"message": "Task " + task.TaskID + " is being processed.",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	total := atomic.LoadInt64(&metrics.TotalRequests)
	totalTime := atomic.LoadInt64(&metrics.AverageProcessingTime)
	active := atomic.LoadInt64(&metrics.ActiveRequests)

	var average int64
	if total > 0 {
		average = totalTime / total
	}

	response := map[string]interface{}{
		"total_requests":          total,
		"average_processing_ms":   average,
		"current_active_requests": active,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or unable to load")
	}

	n, err := strconv.Atoi(os.Getenv("MAX_CONCURRENT_REQUESTS"))
	if err != nil || n <= 0 {
		log.Println("Invalid or missing MAX_CONCURRENT_REQUESTS in .env, defaulting to 10")
		n = 10 // Default value
	}
	log.Printf("Setting max concurrent requests to %d\n", n)

	maxConcurrency = n
	concurrencyLimiter = make(chan struct{}, maxConcurrency)

	mux := http.NewServeMux()
	mux.HandleFunc("/process", processHandler)
	mux.HandleFunc("/metrics", metricsHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Graceful shutdown
	idleConnsClosed := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		log.Println("Shutting down server...")

		// Shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Println("Server is running on :8080")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	<-idleConnsClosed
	log.Println("Server stopped gracefully.")

}
