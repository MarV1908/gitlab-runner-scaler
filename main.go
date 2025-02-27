package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Config struct {
	GitLabToken          string
	RunnerID             string
	PendingJobsPerRunner int
}

type MetricResponse struct {
	MetricName  string `json:"metricName"`
	MetricValue int    `json:"metricValue"`
}

type GitLabJob struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

var config Config

func init() {
	config = Config{
		GitLabToken:          os.Getenv("GITLAB_TOKEN"),
		RunnerID:             os.Getenv("GITLAB_RUNNER_ID"),
		PendingJobsPerRunner: getEnvAsInt("PENDING_JOBS_PER_RUNNER", 10), // Default to 10 jobs per runner
	}
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func fetchPendingJobs() (int, error) {
	url := fmt.Sprintf("https://gitlab.com/api/v4/runners/%s/jobs?status=pending", config.RunnerID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("PRIVATE-TOKEN", config.GitLabToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("GitLab API returned non-200 status: %d", resp.StatusCode)
	}

	var jobs []GitLabJob
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return 0, err
	}

	return len(jobs), nil
}

func calculateDesiredReplicas(pendingJobs int) int {
	if pendingJobs == 0 {
		return 1 // Keep at least 1 runner active
	}
	return (pendingJobs / config.PendingJobsPerRunner) + 1
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	pendingJobs, err := fetchPendingJobs()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch jobs: %v", err), http.StatusInternalServerError)
		return
	}

	desiredReplicas := calculateDesiredReplicas(pendingJobs)
	log.Printf("Pending jobs: %d, Desired runners: %d", pendingJobs, desiredReplicas)

	response := MetricResponse{
		MetricName:  "desired_runners",
		MetricValue: desiredReplicas,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]MetricResponse{response})
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/healthz", readinessHandler)

	port := ":8080"
	log.Printf("Starting external scaler on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
