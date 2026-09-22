package vel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matiasleandrokruk/fenix/internal/domain/evidence"
)

func TestSinkCheckpointAndVerifySupportsWorkspaceSlashStream(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		switch r.URL.Path {
		case "/v1/streams/workspace/ws-1/checkpoints":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"checkpoint_id":"checkpoint-1",
				"stream_id":"workspace/ws-1",
				"tree_size":1,
				"checkpoint_hash":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				"merkle_root":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
			}`))
		case "/v1/streams/workspace/ws-1/bundle":
			_, _ = w.Write([]byte(`{"opaque":"bundle"}`))
		case "/v1/verify":
			_, _ = w.Write([]byte(`{
				"valid":true,
				"stream_id":"workspace/ws-1",
				"checked_events":1,
				"checkpoint_tree_size":1,
				"issues":[]
			}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	sink, err := NewSink(server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("NewSink: %v", err)
	}
	progress, err := sink.CheckpointAndVerify(context.Background(), "workspace/ws-1")
	if err != nil {
		t.Fatalf("CheckpointAndVerify: %v", err)
	}
	if progress.Status != evidence.VerificationVerified || progress.Checkpoint.TreeSize != 1 {
		t.Fatalf("progress = %#v", progress)
	}
}


func TestVerificationProgressCapturesFailureAndIssues(t *testing.T) {
	treeSize := int64(2)
	sequence := int64(2)
	checkpoint := storedCheckpoint{
		CheckpointID:   "checkpoint-2",
		StreamID:       "workspace/ws-1",
		TreeSize:       treeSize,
		CheckpointHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		MerkleRoot:     "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	verification := verificationResponse{
		Valid:              false,
		StreamID:           "workspace/ws-1",
		CheckedEvents:      2,
		CheckpointTreeSize: &treeSize,
		Issues: []verificationIssue{
			{Code: " signature_mismatch ", Sequence: &sequence},
			{Code: "   "},
		},
	}

	progress, err := verificationProgress("workspace/ws-1", checkpoint, verification)
	if err != nil {
		t.Fatalf("verificationProgress: %v", err)
	}
	if progress.Status != evidence.VerificationFailed {
		t.Fatalf("status = %q", progress.Status)
	}
	if len(progress.Issues) != 1 || progress.Issues[0] != "signature_mismatch@2" {
		t.Fatalf("issues = %#v", progress.Issues)
	}
}

func TestVerificationProgressRejectsLifecycleMismatches(t *testing.T) {
	treeSize := int64(2)
	checkpoint := storedCheckpoint{
		CheckpointID:   "checkpoint-2",
		StreamID:       "workspace/ws-1",
		TreeSize:       treeSize,
		CheckpointHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		MerkleRoot:     "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	verification := verificationResponse{
		Valid:              true,
		StreamID:           "workspace/ws-1",
		CheckpointTreeSize: &treeSize,
	}

	wrongStream := checkpoint
	wrongStream.StreamID = "workspace/ws-2"
	if _, err := verificationProgress("workspace/ws-1", wrongStream, verification); err == nil {
		t.Fatal("expected stream identity error")
	}

	wrongSize := int64(1)
	verification.CheckpointTreeSize = &wrongSize
	if _, err := verificationProgress("workspace/ws-1", checkpoint, verification); err == nil {
		t.Fatal("expected checkpoint size mismatch")
	}

	verification.CheckpointTreeSize = &treeSize
	invalidCheckpoint := checkpoint
	invalidCheckpoint.CheckpointID = ""
	if _, err := verificationProgress("workspace/ws-1", invalidCheckpoint, verification); err == nil {
		t.Fatal("expected invalid checkpoint error")
	}
}
