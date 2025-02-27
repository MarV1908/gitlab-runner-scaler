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
	GitLabURL            string
	GitLabToken          string
	PendingJobsPerRunner int
	RunnerTag            string // Tag to search for across all runners
}

type MetricResponse struct {
	MetricName  string `json:"metricName"`
	MetricValue int    `json:"metricValue"`
}

type GitLabJob struct {
	ID     int      `json:"id"`
	Status string   `json:"status"`
	Tags   []string `json:"tag_list"`
}

type GitLabRunner struct {
	ID int `json:"id"`
}

var config Config

func init() {
	// Fetch GitLab URL from the environment, default to GitLab.com if not set
	config = Config{
		GitLabURL:            getEnv("GITLAB_URL", "https://gitlab.com"),
		GitLabToken:          os.Getenv("GITLAB_TOKEN"),
		PendingJobsPerRunner: getEnvAsInt("PENDING_JOBS_PER_RUNNER", 10), // Default to 10 jobs per runner
		RunnerTag:            os.Getenv("GITLAB_RUNNER_TAG"), // Tag for matching runners
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

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Fetch all runners in the GitLab instance
func fetchAllRunners() ([]GitLabRunner, error) {
	url := fmt.Sprintf("%s/api/v4/runners", config.GitLabURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", config.GitLabToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitLab API returned non-200 status: %d", resp.StatusCode)
	}

	var runners []GitLabRunner
	if err := json.NewDecoder(resp.Body).Decode(&runners); err != nil {
		return nil, err
	}

	return runners, nil
}

// Fetch pending jobs for a given runner
func fetchPendingJobsForRunner(runnerID int) ([]GitLabJob, error) {
	url := fmt.Sprintf("%s/api/v4/runners/%d/jobs?status=pending", config.GitLabURL, runnerID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("PRIVATE-TOKEN", config.GitLabToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitLab API returned non-200 status: %d", resp.StatusCode)
	}

	var jobs []GitLabJob
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Fetch pending jobs across all runners with a specific tag
func fetchPendingJobsForAllRunners() (int, error) {
	runners, err := fetchAllRunners()
	if err != nil {
		return 0, err
	}

	totalPendingJobs := 0
	for _, runner := range runners {
		// Fetch jobs for each runner
		jobs, err := fetchPendingJobsForRunner(runner.ID)
		if err != nil {
			log.Printf("Error fetching jobs for runner %d: %v", runner.ID, err)
			continue
		}

		// Filter jobs by tag
		for _, job := range jobs {
			for _, tag := range job.Tags {
				if tag == config.RunnerTag {
					totalPendingJobs++
					break
				}
			}
		}
	}

	return totalPendingJobs, nil
}

// Calculate desired replicas based on pending jobs
func calculateDesiredReplicas(pendingJobs int) int {
	if pendingJobs == 0 {
		return 1 // Keep at least 1 runner active
	}
	return (pendingJobs / config.PendingJobsPerRunner) + 1
}

// Metrics handler
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	pendingJobs, err := fetchPendingJobsForAllRunners()
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

// Readiness handler
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
