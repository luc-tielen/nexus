package todoist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Task represents a Todoist task.
type Task struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

const defaultBaseURL = "https://api.todoist.com/rest/v2"

// doer is the subset of http.Client used for requests.
// Extracted as an interface to allow test doubles.
type doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client sends requests to the Todoist REST API.
type Client struct {
	apiKey  string
	http    doer
	baseURL string
}

// NewClient reads TODOIST_API_KEY from the environment and returns a ready
// client. Returns an error if the variable is unset.
func NewClient() (*Client, error) {
	apiKey := os.Getenv("TODOIST_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("TODOIST_API_KEY is not set")
	}
	return &Client{apiKey: apiKey, http: &http.Client{}, baseURL: defaultBaseURL}, nil
}

// NewClientWithHTTP creates a Client with a custom HTTP doer and base URL.
// Intended for use in tests.
func NewClientWithHTTP(apiKey string, h doer, baseURL string) *Client {
	return &Client{apiKey: apiKey, http: h, baseURL: baseURL}
}

// ListTasks returns all active tasks from Todoist.
func (c *Client) ListTasks() ([]Task, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tasks", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing Todoist tasks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("todoist API returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var tasks []Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("decoding tasks: %w", err)
	}
	return tasks, nil
}

// CompleteTask marks the task with the given ID as completed.
func (c *Client) CompleteTask(taskID string) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tasks/"+taskID+"/close", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("completing Todoist task: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("todoist API returned %s", resp.Status)
	}
	return nil
}

// CreateTask adds a new task with the given content to the Todoist inbox.
func (c *Client) CreateTask(content string) error {
	body, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/tasks", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("creating Todoist task: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("todoist API returned %s", resp.Status)
	}
	return nil
}
