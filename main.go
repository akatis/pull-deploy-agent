package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Environment struct {
	Branch      string `json:"branch"`
	Dir         string `json:"dir"`
	ServiceName string `json:"service_name"`
}

type GitConfig struct {
	Username  string `json:"username"`
	Token     string `json:"token"`
	RepoOwner string `json:"repo_owner"`
	RepoName  string `json:"repo_name"`
	UseAuth   bool   `json:"use_auth"`
}

type SlackConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
}

type Config struct {
	Environments []Environment `json:"environments"`
	LogFile      string        `json:"log_file"`
	Interval     int           `json:"interval_seconds"`
	Git          GitConfig     `json:"git_config"`
	Slack        SlackConfig   `json:"slack"`
}

var config *Config

func main() {
	var err error
	config, err = loadConfig("config.json")
	if err != nil {
		log.Fatalf("Config load error: %v", err)
	}

	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Log file error: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	for {
		for _, env := range config.Environments {
			if err := deploy(env); err != nil {
				log.Printf("[%s] Error: %v\n", env.Branch, err)
				sendSlackMessage(fmt.Sprintf(":x: [%s] Deploy error: %v", env.Branch, err))
			}
		}
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func deploy(env Environment) error {
	log.Printf("[%s] Checking for updates...\n", env.Branch)

	lastCommit, err := gitCommitHash(env.Dir)
	if err != nil {
		return fmt.Errorf("could not get last commit hash: %w", err)
	}

	if err := gitPull(env.Dir, env.Branch); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	newCommit, err := gitCommitHash(env.Dir)
	if err != nil {
		return fmt.Errorf("could not get new commit hash: %w", err)
	}

	if lastCommit == newCommit {
		log.Printf("[%s] No new changes.\n", env.Branch)
		return nil
	}

	log.Printf("[%s] New commit detected. Building...\n", env.Branch)
	binaryPath := filepath.Join(env.Dir, "notify-hub")
	cmd := exec.Command("go", "build", "-o", binaryPath, filepath.Join(env.Dir, "cmd"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %v - %s", err, string(output))
	}

	target := "/usr/local/bin/" + env.ServiceName
	if err := exec.Command("sudo", "cp", binaryPath, target).Run(); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	if err := exec.Command("sudo", "systemctl", "restart", env.ServiceName).Run(); err != nil {
		return fmt.Errorf("systemctl restart failed: %w", err)
	}

	log.Printf("[%s] Deploy completed successfully.\n", env.Branch)
	sendSlackMessage(fmt.Sprintf(":rocket: [%s] Deploy successful at %s", env.Branch, time.Now().Format(time.RFC822)))
	return nil
}

func gitCommitHash(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func gitPull(dir, branch string) error {
	var remoteURL string
	if config.Git.UseAuth {
		remoteURL = fmt.Sprintf("https://%s:%s@github.com/%s/%s.git",
			config.Git.Username,
			config.Git.Token,
			config.Git.RepoOwner,
			config.Git.RepoName)
	} else {
		remoteURL = fmt.Sprintf("https://github.com/%s/%s.git",
			config.Git.RepoOwner,
			config.Git.RepoName)
	}

	setURLCmd := exec.Command("git", "-C", dir, "remote", "set-url", "origin", remoteURL)
	if err := setURLCmd.Run(); err != nil {
		return fmt.Errorf("failed to set remote URL: %w", err)
	}

	cmd := exec.Command("git", "-C", dir, "pull", "origin", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull error: %s", string(output))
	}
	return nil
}

func sendSlackMessage(message string) {
	if !config.Slack.Enabled || config.Slack.WebhookURL == "" {
		return
	}

	payload := map[string]string{"text": message}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", config.Slack.WebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Slack request creation failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Slack post failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Slack returned non-OK status: %v", resp.Status)
	}
}
