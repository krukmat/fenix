package vel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
	"github.com/matiasleandrokruk/fenix/internal/infra/integration/telemetry"
)

type storedCheckpoint struct {
	CheckpointID   string `json:"checkpoint_id"`
	StreamID       string `json:"stream_id"`
	TreeSize       int64  `json:"tree_size"`
	CheckpointHash string `json:"checkpoint_hash"`
	MerkleRoot     string `json:"merkle_root"`
}

type verificationResponse struct {
	Valid              bool                `json:"valid"`
	StreamID           string              `json:"stream_id"`
	CheckedEvents      int                 `json:"checked_events"`
	CheckpointTreeSize *int64              `json:"checkpoint_tree_size"`
	Issues             []verificationIssue `json:"issues"`
}

type verificationIssue struct {
	Code     string `json:"code"`
	Sequence *int64 `json:"sequence"`
}

// CheckpointAndVerify advances one VEL stream without persisting the portable verification bundle.
func (s *Sink) CheckpointAndVerify(
	ctx context.Context,
	streamID string,
) (evidence.VerificationProgress, error) {
	checkpoint, err := s.createCheckpoint(ctx, streamID)
	if err != nil {
		return evidence.VerificationProgress{}, err
	}
	bundleRaw, err := s.fetchBundle(ctx, streamID)
	if err != nil {
		return evidence.VerificationProgress{}, err
	}
	verification, err := s.verifyBundle(ctx, bundleRaw)
	if err != nil {
		return evidence.VerificationProgress{}, err
	}
	return verificationProgress(streamID, checkpoint, verification)
}

func (s *Sink) createCheckpoint(ctx context.Context, streamID string) (storedCheckpoint, error) {
	raw, err := s.executeLifecycleRequest(
		ctx,
		"checkpoint",
		http.MethodPost,
		streamLifecyclePath(streamID, "checkpoints"),
		nil,
	)
	if err != nil {
		return storedCheckpoint{}, err
	}
	var checkpoint storedCheckpoint
	if decodeErr := json.Unmarshal(raw, &checkpoint); decodeErr != nil {
		return storedCheckpoint{}, fmt.Errorf("decode VEL checkpoint: %w", decodeErr)
	}
	return checkpoint, nil
}

func (s *Sink) fetchBundle(ctx context.Context, streamID string) ([]byte, error) {
	return s.executeLifecycleRequest(
		ctx,
		"bundle",
		http.MethodGet,
		streamLifecyclePath(streamID, "bundle"),
		nil,
	)
}

func (s *Sink) verifyBundle(ctx context.Context, bundleRaw []byte) (verificationResponse, error) {
	raw, err := s.executeLifecycleRequest(
		ctx,
		"verify",
		http.MethodPost,
		"/v1/verify",
		bundleRaw,
	)
	if err != nil {
		return verificationResponse{}, err
	}
	var verification verificationResponse
	if decodeErr := json.Unmarshal(raw, &verification); decodeErr != nil {
		return verificationResponse{}, fmt.Errorf("decode VEL verification: %w", decodeErr)
	}
	return verification, nil
}

func (s *Sink) executeLifecycleRequest(
	ctx context.Context,
	operation, method, path string,
	body []byte,
) ([]byte, error) {
	request, err := s.newLifecycleRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	response, err := s.client.Do(request)
	if err != nil {
		telemetry.Default.ObserveProvider(providerNameVEL, operation, providerOutcomeError, time.Since(started))
		return nil, fmt.Errorf("call VEL %s: %w", operation, err)
	}

	raw, err := readLifecycleResponse(operation, response)
	outcome := providerOutcomeSuccess
	if err != nil {
		outcome = providerOutcomeError
	}
	telemetry.Default.ObserveProvider(providerNameVEL, operation, outcome, time.Since(started))
	return raw, err
}

func readLifecycleResponse(operation string, response *http.Response) ([]byte, error) {
	defer response.Body.Close()

	raw, err := readResponse(response)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("VEL %s status %d: %s", operation, response.StatusCode, bodySummary(raw))
	}
	return raw, nil
}

func (s *Sink) newLifecycleRequest(
	ctx context.Context,
	method, path string,
	body []byte,
) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create VEL lifecycle request: %w", err)
	}
	s.decorateRequest(ctx, request)
	return request, nil
}

func streamLifecyclePath(streamID, suffix string) string {
	parts := strings.Split(streamID, "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}
	return "/v1/streams/" + strings.Join(parts, "/") + "/" + suffix
}

func verificationProgress(
	streamID string,
	checkpoint storedCheckpoint,
	verification verificationResponse,
) (evidence.VerificationProgress, error) {
	if checkpoint.StreamID != streamID || verification.StreamID != streamID {
		return evidence.VerificationProgress{}, lifecycleIdentityError(streamID)
	}
	if verification.CheckpointTreeSize != nil && *verification.CheckpointTreeSize != checkpoint.TreeSize {
		return evidence.VerificationProgress{}, errors.New("VEL verification checkpoint size mismatch")
	}
	status := evidence.VerificationFailed
	if verification.Valid {
		status = evidence.VerificationVerified
	}
	progress := evidence.VerificationProgress{
		Checkpoint: evidence.CheckpointReference{
			CheckpointID:   checkpoint.CheckpointID,
			CheckpointHash: checkpoint.CheckpointHash,
			MerkleRoot:     checkpoint.MerkleRoot,
			TreeSize:       checkpoint.TreeSize,
		},
		Status: status,
		Issues: compactVerificationIssues(verification.Issues),
	}
	if err := progress.Validate(); err != nil {
		return evidence.VerificationProgress{}, fmt.Errorf("validate VEL verification progress: %w", err)
	}
	return progress, nil
}

func compactVerificationIssues(issues []verificationIssue) []string {
	result := make([]string, 0, len(issues))
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			continue
		}
		if issue.Sequence != nil {
			code = fmt.Sprintf("%s@%d", code, *issue.Sequence)
		}
		result = append(result, code)
	}
	return result
}

func lifecycleIdentityError(streamID string) error {
	return fmt.Errorf("VEL lifecycle response does not match stream %q", streamID)
}
