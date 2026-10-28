package dubbridge

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

const testTaskID = "550e8400-e29b-41d4-a716-446655440011"

func TestProjectForAgentMinimizesSensitiveData(t *testing.T) {
	queue := &ReviewQueue{OrgID: "550e8400-e29b-41d4-a716-446655440000", ProjectID: "550e8400-e29b-41d4-a716-446655440001", Tasks: []ReviewTask{{ID: testTaskID, State: "pending", AssetID: "550e8400-e29b-41d4-a716-446655440017", AssigneeSubjectID: ptr("550e8400-e29b-41d4-a716-446655440022")}}}
	out, err := ProjectForAgent(queue)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(out.Counts, map[string]int{"pending": 1, "approved": 0, "rejected": 0}) {
		t.Fatalf("counts: %#v", out.Counts)
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "asset_id") || strings.Contains(string(raw), "assignee") || strings.Contains(string(raw), "440017") {
		t.Fatalf("private fields leaked into agent projection")
	}
	if !strings.Contains(string(raw), testTaskID) {
		t.Fatal("missing real task ref")
	}
}
func ptr(s string) *string { return &s }
func TestProjectRejectsInvalidAndDuplicateTask(t *testing.T) {
	for _, queue := range []*ReviewQueue{
		nil,
		{Tasks: nil},
		{Tasks: []ReviewTask{{ID: "malformed", State: "pending"}}},
		{Tasks: []ReviewTask{{ID: testTaskID, State: "unknown"}}},
		{Tasks: []ReviewTask{{ID: testTaskID, State: "pending"}, {ID: testTaskID, State: "approved"}}},
	} {
		if _, err := ProjectForAgent(queue); !errors.Is(err, ErrInvalidResponse) {
			t.Errorf("wanted invalid response for %#v, got %v", queue, err)
		}
	}
}
func TestProjectionAcceptsActualEmptyQueue(t *testing.T) {
	result, err := ProjectForAgent(&ReviewQueue{Tasks: []ReviewTask{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tasks) != 0 || result.Counts["pending"] != 0 {
		t.Fatalf("unexpected empty result %#v", result)
	}
}
func TestTraceHeadersValidated(t *testing.T) {
	for _, value := range []string{"", "bad\nvalue", " bad", strings.Repeat("x", 129), "\x00"} {
		if validCorrelationID(value) {
			t.Errorf("accepted correlation %#v", value)
		}
	}
	if !validCorrelationID("trace-123.AZ:2026") {
		t.Fatal("rejected normal trace")
	}
}
