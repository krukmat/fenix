// Traces: FR-001
package crm

import (
	"testing"
)

func TestCRMHelpers_Private(t *testing.T) {
	// safeFloat64Ptr: nil returns 0
	if got := safeFloat64Ptr(nil); got != 0 {
		t.Fatalf("safeFloat64Ptr(nil) = %v; want 0", got)
	}
	v := 3.14
	if got := safeFloat64Ptr(&v); got != 3.14 {
		t.Fatalf("safeFloat64Ptr(3.14) = %v; want 3.14", got)
	}

	// numberToFloat64: all type branches
	if got := numberToFloat64(float64(1.5)); got != 1.5 {
		t.Fatalf("numberToFloat64(float64) = %v; want 1.5", got)
	}
	if got := numberToFloat64(int64(2)); got != 2.0 {
		t.Fatalf("numberToFloat64(int64) = %v; want 2", got)
	}
	if got := numberToFloat64(int(3)); got != 3.0 {
		t.Fatalf("numberToFloat64(int) = %v; want 3", got)
	}
	if got := numberToFloat64([]byte("4.5")); got != 4.5 {
		t.Fatalf("numberToFloat64([]byte) = %v; want 4.5", got)
	}
	if got := numberToFloat64("5.5"); got != 5.5 {
		t.Fatalf("numberToFloat64(string) = %v; want 5.5", got)
	}
	if got := numberToFloat64(nil); got != 0 {
		t.Fatalf("numberToFloat64(nil/default) = %v; want 0", got)
	}

	// parseOptionalRFC3339: nil returns nil
	if got := parseOptionalRFC3339(nil); got != nil {
		t.Fatalf("parseOptionalRFC3339(nil) should be nil")
	}
	s := "2026-03-10T12:00:00Z"
	got := parseOptionalRFC3339(&s)
	if got == nil {
		t.Fatalf("parseOptionalRFC3339(valid) should not be nil")
	}
	if got.Year() != 2026 {
		t.Fatalf("parseOptionalRFC3339 year = %d; want 2026", got.Year())
	}

	// supportBacklogBucketIndex buckets
	if got := supportBacklogBucketIndex(3); got != 0 {
		t.Fatalf("bucket(3) = %d; want 0", got)
	}
	if got := supportBacklogBucketIndex(7); got != 0 {
		t.Fatalf("bucket(7) = %d; want 0", got)
	}
	if got := supportBacklogBucketIndex(8); got != 1 {
		t.Fatalf("bucket(8) = %d; want 1", got)
	}
	if got := supportBacklogBucketIndex(30); got != 1 {
		t.Fatalf("bucket(30) = %d; want 1", got)
	}
	if got := supportBacklogBucketIndex(31); got != 2 {
		t.Fatalf("bucket(31) = %d; want 2", got)
	}
}
