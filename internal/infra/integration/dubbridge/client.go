// Package dubbridge implements the production HTTP transport for read-only
// Fenix -> DubBridge review queue integration. It never issues credentials.
// The destination is the proposed S2S-only API, not the broad mobile GET path.
package dubbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxResponseBytes     = 2 << 20
	maxQueueItems        = 5000
	maxCorrelationLength = 128
)

var (
	ErrConfiguration      = errors.New("dubbridge: invalid read-only integration configuration")
	ErrUnauthorizedTarget = errors.New("dubbridge: requested resource is not allowlisted")
	ErrInvalidResponse    = errors.New("dubbridge: invalid provider response")
)

// Config defines one fixed authorization target. A caller cannot override
// destination org/project by supplying arbitrary UUIDs in a tool invocation.
type Config struct {
	BaseURL          string
	BearerToken      string // injected only; never committed, logged, or returned
	AllowedOrgID     string
	AllowedProjectID string
	Timeout          time.Duration
}

type Client struct {
	base       *url.URL
	token      string
	org        string
	project    string
	httpClient *http.Client
}

func NewClient(config Config) (*Client, error) {
	if !ValidUUID(config.AllowedOrgID) || !ValidUUID(config.AllowedProjectID) || !validBearer(config.BearerToken) {
		return nil, ErrConfiguration
	}

	base, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || base == nil || base.Hostname() == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, ErrConfiguration
	}
	if base.Scheme != "https" && !(base.Scheme == "http" && isLoopback(base.Hostname())) {
		return nil, ErrConfiguration
	}
	if base.Path != "" && base.Path != "/" && base.Path != "/api" && base.Path != "/api/" {
		return nil, ErrConfiguration
	}
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Second
	}
	if config.Timeout < time.Second || config.Timeout > 60*time.Second {
		return nil, ErrConfiguration
	}
	return &Client{
		base:       base,
		token:      config.BearerToken,
		org:        strings.ToLower(config.AllowedOrgID),
		project:    strings.ToLower(config.AllowedProjectID),
		httpClient: &http.Client{Timeout: config.Timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ValidUUID validates the wire UUID format without allowing path traversal.
func ValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if s[i] != '-' {
				return false
			}
			continue
		}
		b := s[i]
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')) {
			return false
		}
	}
	return true
}

type ReviewTask struct {
	ID                string     `json:"id"`
	OrgID             string     `json:"org_id"`
	ProjectID         string     `json:"project_id"`
	AssetID           string     `json:"asset_id"`
	TargetLanguageID  string     `json:"target_language_id"`
	AssigneeSubjectID *string    `json:"assignee_subject_id"`
	State             string     `json:"state"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	AssignedAt        *time.Time `json:"assigned_at"`
}

type ReviewQueue struct {
	OrgID     string       `json:"org_id"`
	ProjectID string       `json:"project_id"`
	Tasks     []ReviewTask `json:"tasks"`
}

// ProviderError preserves sanitized status and retry disposition, not response
// body or bearer token. Retry only read-only transient failures, bounded by Fenix.
type ProviderError struct {
	Status    int
	Retryable bool
	Err       error
}

func (e *ProviderError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("dubbridge: HTTP %d", e.Status)
	}
	return "dubbridge: provider transport failure"
}
func (e *ProviderError) Unwrap() error { return e.Err }

// ListReviewQueue issues a real GET to DubBridge and validates a genuine
// response. traceID/executionID are correlation hints, not authorization.
func (c *Client) ListReviewQueue(ctx context.Context, orgID, projectID, traceID, executionID string) (*ReviewQueue, error) {
	if !ValidUUID(orgID) || !ValidUUID(projectID) || !strings.EqualFold(orgID, c.org) || !strings.EqualFold(projectID, c.project) {
		return nil, ErrUnauthorizedTarget
	}
	if !validCorrelationID(traceID) || !validCorrelationID(executionID) {
		return nil, ErrConfiguration
	}
	dest := *c.base
	prefix := strings.TrimSuffix(dest.Path, "/")
	dest.Path = fmt.Sprintf("%s/integrations/fenix/orgs/%s/projects/%s/review-tasks", prefix, c.org, c.project)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dest.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Fenix-Trace-ID", traceID)
	req.Header.Set("X-Fenix-Execution-ID", executionID)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &ProviderError{Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ProviderError{Status: resp.StatusCode, Retryable: resp.StatusCode >= 500}
	}
	// Read one byte more than the limit to reject oversize responses.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, &ProviderError{Retryable: true, Err: err}
	}
	if len(raw) > maxResponseBytes {
		return nil, ErrInvalidResponse
	}
	var out ReviewQueue
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, ErrInvalidResponse
	}
	if !strings.EqualFold(out.OrgID, c.org) || !strings.EqualFold(out.ProjectID, c.project) || out.Tasks == nil || len(out.Tasks) > maxQueueItems {
		return nil, ErrInvalidResponse
	}
	seen := make(map[string]struct{}, len(out.Tasks))
	for _, task := range out.Tasks {
		if _, duplicate := seen[strings.ToLower(task.ID)]; duplicate {
			return nil, ErrInvalidResponse
		}
		seen[strings.ToLower(task.ID)] = struct{}{}
		if !ValidUUID(task.ID) || !ValidUUID(task.AssetID) || !ValidUUID(task.TargetLanguageID) || !strings.EqualFold(task.OrgID, c.org) || !strings.EqualFold(task.ProjectID, c.project) {
			return nil, ErrInvalidResponse
		}
		if task.AssigneeSubjectID != nil && !ValidUUID(*task.AssigneeSubjectID) {
			return nil, ErrInvalidResponse
		}
		switch task.State {
		case "pending", "approved", "rejected":
		default:
			return nil, ErrInvalidResponse
		}
		if task.CreatedAt.IsZero() || task.UpdatedAt.IsZero() {
			return nil, ErrInvalidResponse
		}
	}
	return &out, nil
}

// validCorrelationID forbids control characters and unbounded untrusted headers.
func validCorrelationID(value string) bool {
	if len(value) == 0 || len(value) > maxCorrelationLength || strings.TrimSpace(value) != value {
		return false
	}
	for _, c := range value {
		if c < 33 || c > 126 {
			return false
		}
	}
	return true
}

// AgentTask is the deliberately limited observation exposed to Fenix's Blackboard.
// User IDs, asset IDs, language IDs, and all provider credentials are excluded.
type AgentTask struct {
	Ref   string `json:"ref"`
	State string `json:"state"`
}

type AgentResult struct {
	ContractVersion string         `json:"contract_version"`
	Status          string         `json:"status"`
	Counts          map[string]int `json:"counts"`
	Tasks           []AgentTask    `json:"tasks"`
	Source          string         `json:"source"`
}

// ProjectForAgent does not invent tasks and only projects a validated queue.
func ProjectForAgent(queue *ReviewQueue) (AgentResult, error) {
	if queue == nil || queue.Tasks == nil || len(queue.Tasks) > maxQueueItems {
		return AgentResult{}, ErrInvalidResponse
	}
	out := AgentResult{
		ContractVersion: "1",
		Status:          "succeeded",
		Counts:          map[string]int{"pending": 0, "approved": 0, "rejected": 0},
		Tasks:           make([]AgentTask, 0, len(queue.Tasks)),
		Source:          "dubbridge.review_queue",
	}
	seen := make(map[string]struct{}, len(queue.Tasks))
	for _, task := range queue.Tasks {
		if !ValidUUID(task.ID) {
			return AgentResult{}, ErrInvalidResponse
		}
		if _, duplicate := seen[strings.ToLower(task.ID)]; duplicate {
			return AgentResult{}, ErrInvalidResponse
		}
		seen[strings.ToLower(task.ID)] = struct{}{}
		if _, allowed := out.Counts[task.State]; !allowed {
			return AgentResult{}, ErrInvalidResponse
		}
		out.Counts[task.State]++
		out.Tasks = append(out.Tasks, AgentTask{Ref: task.ID, State: task.State})
	}
	return out, nil
}

// validBearer accepts a bounded single HTTP auth-token without whitespace.
// It never inspects credentials semantically or reveals their contents.
func validBearer(value string) bool {
	if len(value) == 0 || len(value) > 8192 {
		return false
	}
	for _, c := range value {
		if c <= 32 || c >= 127 {
			return false
		}
	}
	return true
}
