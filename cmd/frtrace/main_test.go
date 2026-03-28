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
	if tsts[0].BDDFeature != "features/uc-s1-sales-copilot.feature" {
		t.Errorf("expected BDD feature to be loaded, got %s", tsts[0].BDDFeature)
	}
	if tsts[0].BDDBehavior != "" {
		t.Errorf("expected empty BDD behavior in fixture, got %s", tsts[0].BDDBehavior)
	}
}

func TestLoadDoorstopUCs(t *testing.T) {
	ucs, err := loadDoorstopUCs(filepath.Join("testdata", "reqs", "UC"))
	if err != nil {
		t.Fatalf("loadDoorstopUCs: %v", err)
	}
	if len(ucs) != len(requiredUCIDs) {
		t.Fatalf("expected %d UCs, got %d", len(requiredUCIDs), len(ucs))
	}
	if !ucs["UC_S1"].Active {
		t.Error("UC_S1 should be active")
	}
	if ucs["UC_A2"].BehaviorFamily != "define_workflow*" {
		t.Errorf("expected UC_A2 behavior family to be loaded, got %s", ucs["UC_A2"].BehaviorFamily)
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
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{
		ID:          "TST_TEST1",
		Ref:         "src/good_test.go",
		FRLinks:     []string{"FR_TEST1"},
		BDDFeature:  "features/uc-s1-sales-copilot.feature",
		BDDScenario: "Launch Sales Copilot from account detail with grounded context",
		BDDStack:    "mobile",
	}}
	features := map[string]FeatureSpec{
		"features/uc-s1-sales-copilot.feature": {
			Path: "features/uc-s1-sales-copilot.feature",
			Scenarios: map[string]FeatureScenario{
				"Launch Sales Copilot from account detail with grounded context": {
					Name:  "Launch Sales Copilot from account detail with grounded context",
					Tags:  []string{"@UC-S1", "@stack-mobile", "@FR-TEST1", "@TST-TEST1"},
					Stack: "mobile",
				},
			},
		},
	}
	fileTraces := map[string][]string{"src/good_test.go": {"FR-TEST1"}}
	if violations := validate(frs, ucs, tsts, features, fileTraces, "testdata"); len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %v", violations)
	}
}

func TestValidate_UncoveredFR(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}, "FR_TEST3": {Active: true}}
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/good_test.go", FRLinks: []string{"FR_TEST1"}}}
	fileTraces := map[string][]string{"src/good_test.go": {"FR-TEST1"}}
	violations := validate(frs, ucs, tsts, nil, fileTraces, "testdata")
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
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/bad_test.go", FRLinks: []string{"FR_TEST1"}}}
	fileTraces := map[string][]string{"src/bad_test.go": {}}
	violations := validate(frs, ucs, tsts, nil, fileTraces, "testdata")
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
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/good_test.go", FRLinks: []string{"FR_TEST1"}}}
	fileTraces := map[string][]string{"src/good_test.go": {"FR-TEST1", "FR-UNKNOWN"}}
	violations := validate(frs, ucs, tsts, nil, fileTraces, "testdata")
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
	ucs := buildRequiredUCMap("FR_TEST2")
	if violations := validate(frs, ucs, nil, nil, map[string][]string{}, "testdata"); len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidate_FileNotFound(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/nonexistent_test.go", FRLinks: []string{"FR_TEST1"}}}
	violations := validate(frs, ucs, tsts, nil, map[string][]string{}, "testdata")
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

func TestValidate_BadUCLink(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	ucs := buildRequiredUCMap("FR_TEST1")
	ucs["UC_S1"] = UCItem{Active: true, FRLinks: []string{"FR_UNKNOWN"}}

	violations := validate(frs, ucs, nil, nil, map[string][]string{}, "testdata")
	ok := false
	for _, v := range violations {
		if v.Code == "UC-BAD-FR-LINK" {
			ok = true
		}
	}
	if !ok {
		t.Fatal("expected UC-BAD-FR-LINK")
	}
}

func buildRequiredUCMap(frID string) map[string]UCItem {
	ucs := make(map[string]UCItem, len(requiredUCIDs))
	for _, id := range requiredUCIDs {
		ucs[id] = UCItem{Active: true, FRLinks: []string{frID}}
	}
	return ucs
}

func TestLoadFeatureSpecs(t *testing.T) {
	features, err := loadFeatureSpecs(filepath.Join("testdata", "features"), "testdata")
	if err != nil {
		t.Fatalf("loadFeatureSpecs: %v", err)
	}
	spec, ok := features["features/uc-s1-sales-copilot.feature"]
	if !ok {
		t.Fatal("expected test feature to be loaded")
	}
	scenario, ok := spec.Scenarios["Launch Sales Copilot from account detail with grounded context"]
	if !ok {
		t.Fatal("expected scenario to be loaded")
	}
	if scenario.Stack != "mobile" {
		t.Fatalf("expected stack mobile, got %s", scenario.Stack)
	}
}

func TestValidate_BDDScenarioMismatch(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{
		ID:          "TST_TEST1",
		Ref:         "src/good_test.go",
		FRLinks:     []string{"FR_TEST1"},
		BDDFeature:  "features/uc-s1-sales-copilot.feature",
		BDDScenario: "Unknown Scenario",
		BDDStack:    "mobile",
	}}
	features := map[string]FeatureSpec{
		"features/uc-s1-sales-copilot.feature": {
			Path: "features/uc-s1-sales-copilot.feature",
			Scenarios: map[string]FeatureScenario{
				"Launch Sales Copilot from account detail with grounded context": {
					Name:  "Launch Sales Copilot from account detail with grounded context",
					Tags:  []string{"@UC-S1", "@stack-mobile", "@FR-TEST1", "@TST-TEST1"},
					Stack: "mobile",
				},
			},
		},
	}
	violations := validate(frs, ucs, tsts, features, map[string][]string{"src/good_test.go": {"FR-TEST1"}}, "testdata")
	if !hasViolationCode(violations, "BDD-SCENARIO-NOT-FOUND") {
		t.Fatalf("expected BDD-SCENARIO-NOT-FOUND, got %v", violations)
	}
}

func TestValidate_BDDTSTTagMismatch(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{
		ID:          "TST_TEST1",
		Ref:         "src/good_test.go",
		FRLinks:     []string{"FR_TEST1"},
		BDDFeature:  "features/uc-s1-sales-copilot.feature",
		BDDScenario: "Launch Sales Copilot from account detail with grounded context",
		BDDStack:    "mobile",
	}}
	features := map[string]FeatureSpec{
		"features/uc-s1-sales-copilot.feature": {
			Path: "features/uc-s1-sales-copilot.feature",
			Scenarios: map[string]FeatureScenario{
				"Launch Sales Copilot from account detail with grounded context": {
					Name:  "Launch Sales Copilot from account detail with grounded context",
					Tags:  []string{"@UC-S1", "@stack-mobile", "@TST-OTHER"},
					Stack: "mobile",
				},
			},
		},
	}
	violations := validate(frs, ucs, tsts, features, map[string][]string{"src/good_test.go": {"FR-TEST1"}}, "testdata")
	if !hasViolationCode(violations, "BDD-TST-TAG-MISMATCH") {
		t.Fatalf("expected BDD-TST-TAG-MISMATCH, got %v", violations)
	}
}

func TestValidate_BDDInvalidFeatureTags(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/good_test.go", FRLinks: []string{"FR_TEST1"}}}
	features := map[string]FeatureSpec{
		"features/invalid.feature": {
			Path: "features/invalid.feature",
			Scenarios: map[string]FeatureScenario{
				"Invalid tags": {
					Name:  "Invalid tags",
					Tags:  []string{"@UC-UNKNOWN", "@FR-UNKNOWN", "@TST-UNKNOWN", "@stack-go"},
					Stack: "go",
				},
			},
		},
	}
	violations := validate(frs, ucs, tsts, features, map[string][]string{"src/good_test.go": {"FR-TEST1"}}, "testdata")
	if !hasViolationCode(violations, "BDD-UC-TAG-INVALID") {
		t.Fatalf("expected BDD-UC-TAG-INVALID, got %v", violations)
	}
	if !hasViolationCode(violations, "BDD-FR-TAG-INVALID") {
		t.Fatalf("expected BDD-FR-TAG-INVALID, got %v", violations)
	}
	if !hasViolationCode(violations, "BDD-TST-TAG-INVALID") {
		t.Fatalf("expected BDD-TST-TAG-INVALID, got %v", violations)
	}
}

func TestValidate_BDDUCTagCount(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	ucs := buildRequiredUCMap("FR_TEST1")
	tsts := []TSTItem{{ID: "TST_TEST1", Ref: "src/good_test.go", FRLinks: []string{"FR_TEST1"}}}
	features := map[string]FeatureSpec{
		"features/invalid.feature": {
			Path: "features/invalid.feature",
			Scenarios: map[string]FeatureScenario{
				"Missing UC tag": {
					Name:  "Missing UC tag",
					Tags:  []string{"@FR-TEST1", "@TST-TEST1", "@stack-go"},
					Stack: "go",
				},
			},
		},
	}
	violations := validate(frs, ucs, tsts, features, map[string][]string{"src/good_test.go": {"FR-TEST1"}}, "testdata")
	if !hasViolationCode(violations, "BDD-UC-TAG-COUNT") {
		t.Fatalf("expected BDD-UC-TAG-COUNT, got %v", violations)
	}
}

func TestValidate_BDDBehaviorTagMismatch(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}}
	ucs := buildRequiredUCMap("FR_TEST1")
	ucs["UC_A2"] = UCItem{Active: true, BehaviorFamily: "define_workflow*", FRLinks: []string{"FR_TEST1"}}
	tsts := []TSTItem{{
		ID:          "TST_TEST1",
		Ref:         "src/good_test.go",
		FRLinks:     []string{"FR_TEST1"},
		BDDFeature:  "features/agent.feature",
		BDDScenario: "Behavior mismatch",
		BDDStack:    "go",
		BDDBehavior: "define_workflow",
	}}
	features := map[string]FeatureSpec{
		"features/agent.feature": {
			Path: "features/agent.feature",
			Scenarios: map[string]FeatureScenario{
				"Behavior mismatch": {
					Name:  "Behavior mismatch",
					Tags:  []string{"@UC-A2", "@stack-go", "@TST-TEST1", "@FR-TEST1", "@behavior-verify_workflow"},
					Stack: "go",
				},
			},
		},
	}
	violations := validate(frs, ucs, tsts, features, map[string][]string{"src/good_test.go": {"FR-TEST1"}}, "testdata")
	if !hasViolationCode(violations, "BDD-BEHAVIOR-TAG-INVALID") {
		t.Fatalf("expected BDD-BEHAVIOR-TAG-INVALID, got %v", violations)
	}
	if !hasViolationCode(violations, "BDD-BEHAVIOR-TAG-MISMATCH") {
		t.Fatalf("expected BDD-BEHAVIOR-TAG-MISMATCH, got %v", violations)
	}
}

func TestValidate_BDDFRLinkMismatch(t *testing.T) {
	frs := map[string]FRItem{"FR_TEST1": {Active: true}, "FR_TEST2": {Active: false}}
	ucs := buildRequiredUCMap("FR_TEST1")
	ucs["UC_A2"] = UCItem{Active: true, BehaviorFamily: "define_workflow*", FRLinks: []string{"FR_TEST1"}}
	tsts := []TSTItem{{
		ID:          "TST_TEST1",
		Ref:         "src/good_test.go",
		FRLinks:     []string{"FR_TEST1"},
		BDDFeature:  "features/agent.feature",
		BDDScenario: "FR mismatch",
		BDDStack:    "go",
		BDDBehavior: "define_workflow",
	}}
	features := map[string]FeatureSpec{
		"features/agent.feature": {
			Path: "features/agent.feature",
			Scenarios: map[string]FeatureScenario{
				"FR mismatch": {
					Name:  "FR mismatch",
					Tags:  []string{"@UC-A2", "@stack-go", "@TST-TEST1", "@FR-TEST2", "@behavior-define_workflow"},
					Stack: "go",
				},
			},
		},
	}
	violations := validate(frs, ucs, tsts, features, map[string][]string{"src/good_test.go": {"FR-TEST1"}}, "testdata")
	if !hasViolationCode(violations, "BDD-FR-LINK-MISMATCH") {
		t.Fatalf("expected BDD-FR-LINK-MISMATCH, got %v", violations)
	}
}

func hasViolationCode(violations []Violation, code string) bool {
	for _, violation := range violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}

func TestMain(m *testing.M) {
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		_ = os.Chdir("cmd/frtrace")
	}
	os.Exit(m.Run())
}
