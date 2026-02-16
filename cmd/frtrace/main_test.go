package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDoorstopFRs(t *testing.T) {
	frs, err := loadDoorstopFRs(filepath.Join("testdata", "reqs", "FR"))
	if err != nil {
		t.Fatalf("loadDoorstopFRs: %v", err)
	}
	if len(frs) != 2 {
		t.Fatalf("expected 2 FRs, got %d", len(frs))
	}
	if !frs["FR_TEST1"].Active {
		t.Error("FR_TEST1 should be active")
	}
	if frs["FR_TEST2"].Active {
		t.Error("FR_TEST2 should be inactive")
	}
}

func TestLoadDoorstopTSTs(t *testing.T) {
	tsts, err := loadDoorstopTSTs(filepath.Join("testdata", "reqs", "TST"))
	if err != nil {
		t.Fatalf("loadDoorstopTSTs: %v", err)
	}
	if len(tsts) != 1 {
		t.Fatalf("expected 1 TST, got %d", len(tsts))
	}
	if tsts[0].Ref != "src/good_test.go" {
		t.Errorf("expected ref src/good_test.go, got %s", tsts[0].Ref)
	}
}

func TestScanTraces(t *testing.T) {
	traces, err := scanTraces(filepath.Join("testdata", "src", "good_test.go"))
	if err != nil {
		t.Fatalf("scanTraces: %v", err)
	}
	if len(traces) != 1 || traces[0] != "FR-TEST1" {
		t.Errorf("unexpected traces: %v", traces)
	}
}

func TestValidate_AllCovered(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/good_test.go", FRLinks: []string{"FR_TEST1"}}}
	fileTraces := map[string][]string{"src/good_test.go": {"FR-TEST1"}}
	if violations := validate(frs, tsts, fileTraces, "testdata"); len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %v", violations)
	}
}

func TestValidate_UncoveredFR(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}, "FR_TEST3": {Active: true}}
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/good_test.go", FRLinks: []string{"FR_TEST1"}}}
	fileTraces := map[string][]string{"src/good_test.go": {"FR-TEST1"}}
	violations := validate(frs, tsts, fileTraces, "testdata")
	ok := false
	for _, v := range violations {
		if v.Code == "UNCOVERED" && v.FRID == "FR_TEST3" {
			ok = true
		}
	}
	if !ok {
		t.Fatal("expected UNCOVERED for FR_TEST3")
	}
}

func TestValidate_MissingAnnotation(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/bad_test.go", FRLinks: []string{"FR_TEST1"}}}
	fileTraces := map[string][]string{"src/bad_test.go": {}}
	violations := validate(frs, tsts, fileTraces, "testdata")
	ok := false
	for _, v := range violations {
		if v.Code == "MISSING-ANNOTATION" {
			ok = true
		}
	}
	if !ok {
		t.Fatal("expected MISSING-ANNOTATION")
	}
}

func TestValidate_OrphanAnnotation(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/good_test.go", FRLinks: []string{"FR_TEST1"}}}
	fileTraces := map[string][]string{"src/good_test.go": {"FR-TEST1", "FR-UNKNOWN"}}
	violations := validate(frs, tsts, fileTraces, "testdata")
	ok := false
	for _, v := range violations {
		if v.Code == "ORPHAN" {
			ok = true
		}
	}
	if !ok {
		t.Fatal("expected ORPHAN")
	}
}

func TestValidate_InactiveFRSkipped(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST2": {Active: false}}
	if violations := validate(frs, nil, map[string][]string{}, "testdata"); len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidate_FileNotFound(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/nonexistent_test.go", FRLinks: []string{"FR_TEST1"}}}
	violations := validate(frs, tsts, map[string][]string{}, "testdata")
	ok := false
	for _, v := range violations {
		if v.Code == "FILE-NOT-FOUND" {
			ok = true
		}
	}
	if !ok {
		t.Fatal("expected FILE-NOT-FOUND")
	}
}

func TestMain(m *testing.M) {
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		_ = os.Chdir("cmd/frtrace")
	}
	os.Exit(m.Run())
}
