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
