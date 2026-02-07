package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// Airflow 3 API v2 endpoints
const (
	EndpointDAGs          = "/api/v2/dags"
	EndpointDAGRuns       = "/api/v2/dags/%s/dagRuns"
	EndpointTaskInstances = "/api/v2/dags/%s/dagRuns/%s/taskInstances"
	EndpointTasks         = "/api/v2/dags/%s/tasks"
	EndpointTaskLogs      = "/api/v2/dags/%s/dagRuns/%s/taskInstances/%s/logs/%d"
	EndpointHealth        = "/api/v2/monitor/health"
	EndpointDAGSource     = "/api/v2/dagSources/%s"
	EndpointConfig        = "/api/v2/config"
	EndpointConnections   = "/api/v2/connections"
	EndpointVariables     = "/api/v2/variables"
	EndpointAuthToken     = "/auth/token"
	EndpointBackfills     = "/api/v2/backfills"
)

type Client struct {
	baseURL     string
	httpClient  *http.Client
	username    string
	password    string
	rateLimiter <-chan time.Time

	// JWT token management
	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

type ClientConfig struct {
	BaseURL  string
	Username string
	Password string
	Token    string // pre-existing token (optional)
	AuthType string // ignored in Airflow 3 (always JWT)
	Timeout  time.Duration
}

func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	c := &Client{
		baseURL:  cfg.BaseURL,
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		rateLimiter: time.Tick(100 * time.Millisecond), //nolint:staticcheck
	}

	// If a static token was provided, use it
	if cfg.Token != "" {
		c.accessToken = cfg.Token
		c.tokenExpiry = time.Now().Add(24 * time.Hour)
	}

	return c
}

// ListOptions controls pagination and ordering for list endpoints.
type ListOptions struct {
	Limit   int
	Offset  int
	OrderBy string
}

func (o *ListOptions) apply(q url.Values) {
	if o == nil {
		return
	}
	if o.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", o.Limit))
	}
	if o.Offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", o.Offset))
	}
	if o.OrderBy != "" {
		q.Set("order_by", o.OrderBy)
	}
}

// ---------- JWT Authentication ----------

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

// ensureToken acquires or refreshes a JWT token if needed.
func (c *Client) ensureToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reuse existing token if still valid (with 60s buffer)
	if c.accessToken != "" && time.Now().Add(60*time.Second).Before(c.tokenExpiry) {
		return nil
	}

	if c.username == "" {
		return fmt.Errorf("no credentials configured")
	}

	body, _ := json.Marshal(map[string]string{
		"username": c.username,
		"password": c.password,
	})

	req, err := http.NewRequest("POST", c.baseURL+EndpointAuthToken, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed %s: %s", resp.Status, string(respBody))
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return fmt.Errorf("decode token: %w", err)
	}

	c.accessToken = tok.AccessToken
	// Airflow 3 default token lifetime is 24h; use 23h to be safe
	c.tokenExpiry = time.Now().Add(23 * time.Hour)
	return nil
}

// ---------- DAGs ----------

func (c *Client) GetDAGs(ctx context.Context, opts *ListOptions) (*models.DAGCollection, error) {
	var out models.DAGCollection
	if err := c.get(ctx, EndpointDAGs, opts, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- DAG Runs ----------

func (c *Client) GetDAGRuns(ctx context.Context, dagId string, opts *ListOptions) (*models.DAGRunCollection, error) {
	var out models.DAGRunCollection
	endpoint := fmt.Sprintf(EndpointDAGRuns, dagId)
	if err := c.get(ctx, endpoint, opts, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Tasks (DAG structure) ----------

func (c *Client) GetTasks(ctx context.Context, dagId string) (*models.TaskCollection, error) {
	var out models.TaskCollection
	endpoint := fmt.Sprintf(EndpointTasks, dagId)
	if err := c.get(ctx, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Task Instances ----------

func (c *Client) GetTaskInstances(ctx context.Context, dagId, runId string, opts *ListOptions) (*models.TaskInstanceCollection, error) {
	var out models.TaskInstanceCollection
	endpoint := fmt.Sprintf(EndpointTaskInstances, dagId, runId)
	if err := c.get(ctx, endpoint, opts, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Task Logs ----------

// GetTaskLogs fetches logs for a task instance. tryNumber defaults to 1 if <= 0.
func (c *Client) GetTaskLogs(ctx context.Context, dagId, runId, taskId string, tryNumber int) (string, error) {
	<-c.rateLimiter

	if err := c.ensureToken(); err != nil {
		return "", err
	}

	if tryNumber <= 0 {
		tryNumber = 1
	}

	endpoint := fmt.Sprintf(EndpointTaskLogs, dagId, runId, taskId, tryNumber)
	reqURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.readError(resp)
	}

	// Airflow 3 returns JSON: {"content":[{"event":"...","timestamp":"..."}, ...]}
	var logResp struct {
		Content []struct {
			Event     string `json:"event"`
			Timestamp string `json:"timestamp"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&logResp); err != nil {
		return "", fmt.Errorf("decode log response: %w", err)
	}

	var result string
	for _, entry := range logResp.Content {
		if entry.Timestamp != "" {
			result += fmt.Sprintf("[%s] %s\n", entry.Timestamp, entry.Event)
		} else {
			result += entry.Event + "\n"
		}
	}
	return result, nil
}

// ---------- Health ----------

func (c *Client) GetHealth(ctx context.Context) (*models.HealthInfo, error) {
	var out models.HealthInfo
	if err := c.get(ctx, EndpointHealth, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- DAG Source ----------

func (c *Client) GetDAGSource(ctx context.Context, dagId string) (string, error) {
	<-c.rateLimiter

	if err := c.ensureToken(); err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf(EndpointDAGSource, dagId)
	reqURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Accept", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", c.readError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(body), nil
}

// ---------- Config ----------

func (c *Client) GetConfig(ctx context.Context) (*models.AirflowConfigResponse, error) {
	var out models.AirflowConfigResponse
	if err := c.get(ctx, EndpointConfig, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Connections ----------

func (c *Client) GetConnections(ctx context.Context, opts *ListOptions) (*models.ConnectionCollection, error) {
	var out models.ConnectionCollection
	if err := c.get(ctx, EndpointConnections, opts, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Variables ----------

func (c *Client) GetVariables(ctx context.Context, opts *ListOptions) (*models.VariableCollection, error) {
	var out models.VariableCollection
	if err := c.get(ctx, EndpointVariables, opts, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- DAG Operations ----------

func (c *Client) TriggerDAGRun(ctx context.Context, dagId string, body map[string]any) (*models.DAGRun, error) {
	var out models.DAGRun
	endpoint := fmt.Sprintf(EndpointDAGRuns, dagId)
	if err := c.post(ctx, endpoint, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PauseDAG(ctx context.Context, dagId string) error {
	return c.patch(ctx, fmt.Sprintf(EndpointDAGs+"/%s", dagId), map[string]any{"is_paused": true}, nil)
}

func (c *Client) UnpauseDAG(ctx context.Context, dagId string) error {
	return c.patch(ctx, fmt.Sprintf(EndpointDAGs+"/%s", dagId), map[string]any{"is_paused": false}, nil)
}

func (c *Client) CreateBackfill(ctx context.Context, body map[string]any) (*models.BackfillResponse, error) {
	var out models.BackfillResponse
	if err := c.post(ctx, EndpointBackfills, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- internal helpers ----------

func (c *Client) post(ctx context.Context, endpoint string, body any, out any) error {
	<-c.rateLimiter

	if err := c.ensureToken(); err != nil {
		return err
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.readError(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) patch(ctx context.Context, endpoint string, body any, out any) error {
	<-c.rateLimiter

	if err := c.ensureToken(); err != nil {
		return err
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", c.baseURL+endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.readError(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, endpoint string, opts *ListOptions, out any) error {
	<-c.rateLimiter

	if err := c.ensureToken(); err != nil {
		return err
	}

	reqURL, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	q := reqURL.Query()
	opts.apply(q)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	c.setAuth(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.readError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) setAuth(req *http.Request) {
	c.mu.Lock()
	token := c.accessToken
	c.mu.Unlock()

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func (c *Client) readError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("api error %s: %s", resp.Status, string(body))
}
