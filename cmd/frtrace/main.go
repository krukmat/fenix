// Task Gateway: FR-to-test traceability scanner.
// Reads Doorstop YAML files and validates // Traces: annotations in Go test files.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type FRItem struct {
	Active bool `yaml:"active"`
}

type UCItem struct {
	Active         bool   `yaml:"active"`
	BehaviorFamily string `yaml:"behavior_family"`
	FRLinks        []string
}

type TSTItem struct {
	ID          string
	Ref         string
	FRLinks     []string
	BDDFeature  string
	BDDScenario string
	BDDStack    string
	BDDBehavior string
}

type ucYAML struct {
	Active         bool        `yaml:"active"`
	BehaviorFamily string      `yaml:"behavior_family"`
	Links          interface{} `yaml:"links"`
}

type bddYAML struct {
	Feature  string `yaml:"feature"`
	Scenario string `yaml:"scenario"`
	Stack    string `yaml:"stack"`
	Behavior string `yaml:"behavior"`
}

type tstYAML struct {
	Active bool        `yaml:"active"`
	Ref    string      `yaml:"ref"`
	Links  interface{} `yaml:"links"`
	BDD    bddYAML     `yaml:"bdd"`
}

type FeatureScenario struct {
	Name  string
	Tags  []string
	Stack string
}

type FeatureSpec struct {
	Path      string
	Scenarios map[string]FeatureScenario
}

type Violation struct {
	Code    string
	FRID    string
	TSTID   string
	File    string
	Message string
}

const (
	flagReqs        = "reqs"
	defaultReqsDir  = "./reqs"
	extYAML         = ".yml"
	doorstopYAML    = ".doorstop.yml"
	errReadFileFmt  = "reading %s: %w"
	errParseFileFmt = "parsing %s: %w"
)

var requiredUCIDs = []string{
	"UC_S1", "UC_S2", "UC_S3", "UC_C1", "UC_K1", "UC_D1", "UC_G1", "UC_A1",
	"UC_A2", "UC_A3", "UC_A4", "UC_A5", "UC_A6", "UC_A7", "UC_A8", "UC_A9",
}

var compactDoorstopIDRegex = regexp.MustCompile(`^([A-Z]+)(\d+)$`)

func main() {
	reqsDir := flag.String(flagReqs, defaultReqsDir, "Path to Doorstop requirements directory")
	rootDir := flag.String("root", ".", "Project root directory")
	flag.Parse()

	frs, err := loadDoorstopFRs(filepath.Join(*reqsDir, "FR"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading FRs: %v\n", err)
		os.Exit(1)
	}
	ucs, err := loadDoorstopUCs(filepath.Join(*reqsDir, "UC"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading UCs: %v\n", err)
		os.Exit(1)
	}
	tsts, err := loadDoorstopTSTs(filepath.Join(*reqsDir, "TST"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading TSTs: %v\n", err)
		os.Exit(1)
	}
	features, err := loadFeatureSpecs(filepath.Join(*rootDir, "features"), *rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading features: %v\n", err)
		os.Exit(1)
	}

	fileTraces := buildFileTraces(tsts, *rootDir)
	violations := validate(frs, ucs, tsts, features, fileTraces, *rootDir)
	printReport(frs, ucs, tsts, features, fileTraces, violations)
}

func buildFileTraces(tsts []TSTItem, rootDir string) map[string][]string {
	fileTraces := make(map[string][]string)
	for _, tst := range tsts {
		fullPath := filepath.Join(rootDir, tst.Ref)
		if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
			fileTraces[tst.Ref] = nil
			continue
		}
		traces, scanErr := scanTraces(fullPath)
		if scanErr != nil {
			fmt.Fprintf(os.Stderr, "WARNING scanning %s: %v\n", tst.Ref, scanErr)
			continue
		}
		fileTraces[tst.Ref] = traces
	}
	return fileTraces
}

func printReport(frs map[string]FRItem, ucs map[string]UCItem, tsts []TSTItem, features map[string]FeatureSpec, fileTraces map[string][]string, violations []Violation) {
	fmt.Printf("=== FR Traceability Report ===\n")
	fmt.Printf("FRs loaded: %d (active: %d)\n", len(frs), countActive(frs))
	fmt.Printf("UCs loaded: %d (active: %d)\n", len(ucs), countActiveUCs(ucs))
	fmt.Printf("TST items loaded: %d\n", len(tsts))
	fmt.Printf("Feature files loaded: %d\n", len(features))
	fmt.Printf("Test files scanned: %d\n", len(fileTraces))
	fmt.Printf("Violations: %d\n\n", len(violations))
	for _, v := range violations {
		fmt.Printf("[%s] %s\n", v.Code, v.Message)
	}
	if len(violations) > 0 {
		fmt.Printf("\nFAILED: %d traceability violations found\n", len(violations))
		os.Exit(1)
	}
	fmt.Println("\nPASSED: all active FRs have traced tests")
}

func shouldSkipDoorstopEntry(entry os.DirEntry) bool {
	return entry.IsDir() || !strings.HasSuffix(entry.Name(), extYAML) || entry.Name() == doorstopYAML
}

func loadDoorstopFRs(dir string) (map[string]FRItem, error) {
	frs := make(map[string]FRItem)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading FR directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if shouldSkipDoorstopEntry(entry) {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf(errReadFileFmt, entry.Name(), readErr)
		}
		var fr FRItem
		if parseErr := yaml.Unmarshal(data, &fr); parseErr != nil {
			return nil, fmt.Errorf(errParseFileFmt, entry.Name(), parseErr)
		}
		id := strings.TrimSuffix(entry.Name(), extYAML)
		frs[id] = fr
	}
	return frs, nil
}

func loadDoorstopUCs(dir string) (map[string]UCItem, error) {
	ucs := make(map[string]UCItem)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading UC directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if shouldSkipDoorstopEntry(entry) {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf(errReadFileFmt, entry.Name(), readErr)
		}
		var raw ucYAML
		if parseErr := yaml.Unmarshal(data, &raw); parseErr != nil {
			return nil, fmt.Errorf(errParseFileFmt, entry.Name(), parseErr)
		}
		id := strings.TrimSuffix(entry.Name(), extYAML)
		ucs[id] = UCItem{
			Active:         raw.Active,
			BehaviorFamily: raw.BehaviorFamily,
			FRLinks:        extractFRLinks(raw.Links),
		}
	}
	return ucs, nil
}

func loadDoorstopTSTs(dir string) ([]TSTItem, error) {
	var tsts []TSTItem
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading TST directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if shouldSkipDoorstopEntry(entry) {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf(errReadFileFmt, entry.Name(), readErr)
		}
		var raw tstYAML
		if parseErr := yaml.Unmarshal(data, &raw); parseErr != nil {
			return nil, fmt.Errorf(errParseFileFmt, entry.Name(), parseErr)
		}
		id := strings.TrimSuffix(entry.Name(), extYAML)
		tsts = append(tsts, TSTItem{
			ID:          id,
			Ref:         raw.Ref,
			FRLinks:     extractFRLinks(raw.Links),
			BDDFeature:  raw.BDD.Feature,
			BDDScenario: raw.BDD.Scenario,
			BDDStack:    raw.BDD.Stack,
			BDDBehavior: raw.BDD.Behavior,
		})
	}
	return tsts, nil
}

func loadFeatureSpecs(dir string, rootDir string) (map[string]FeatureSpec, error) {
	features := make(map[string]FeatureSpec)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return features, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading features directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("features path %s is not a directory", dir)
	}
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".feature") {
			return nil
		}
		spec, parseErr := parseFeatureFile(path, rootDir)
		if parseErr != nil {
			return parseErr
		}
		features[spec.Path] = spec
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walking features directory %s: %w", dir, walkErr)
	}
	return features, nil
}

func parseFeatureFile(path string, rootDir string) (FeatureSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FeatureSpec{}, fmt.Errorf(errReadFileFmt, path, err)
	}
	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		return FeatureSpec{}, fmt.Errorf("computing feature path for %s: %w", path, err)
	}
	spec := FeatureSpec{
		Path:      filepath.ToSlash(relPath),
		Scenarios: make(map[string]FeatureScenario),
	}
	var featureTags []string
	var pendingTags []string
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "@") {
			pendingTags = append(pendingTags, strings.Fields(line)...)
			continue
		}
		if strings.HasPrefix(line, "Feature:") {
			featureTags = append([]string(nil), pendingTags...)
			pendingTags = nil
			continue
		}
		if !strings.HasPrefix(line, "Scenario:") {
			pendingTags = nil
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, "Scenario:"))
		scenarioTags := append(append([]string(nil), featureTags...), pendingTags...)
		scenario := FeatureScenario{
			Name:  name,
			Tags:  scenarioTags,
			Stack: extractStackTag(scenarioTags),
		}
		spec.Scenarios[name] = scenario
		pendingTags = nil
	}
	return spec, nil
}

func extractStackTag(tags []string) string {
	for _, tag := range tags {
		switch tag {
		case "@stack-go":
			return "go"
		case "@stack-bff":
			return "bff"
		case "@stack-mobile":
			return "mobile"
		}
	}
	return ""
}

func extractFRLinks(links interface{}) []string {
	var result []string
	linkSlice, ok := links.([]interface{})
	if !ok {
		return result
	}
	for _, item := range linkSlice {
		switch v := item.(type) {
		case string:
			result = append(result, normalizeDoorstopLinkID(v))
		case map[string]interface{}:
			for key := range v {
				result = append(result, normalizeDoorstopLinkID(key))
			}
		}
	}
	return result
}

func normalizeDoorstopLinkID(id string) string {
	if strings.Contains(id, "_") {
		return id
	}
	matches := compactDoorstopIDRegex.FindStringSubmatch(id)
	if len(matches) != 3 {
		return id
	}
	return matches[1] + "_" + matches[2]
}

var tracesRegex = regexp.MustCompile(`//\s*Traces:\s*(.+)`)

func scanTraces(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var traces []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineTraces := extractTraceAnnotation(scanner.Text())
		traces = append(traces, lineTraces...)
	}
	return traces, scanner.Err()
}

// extractTraceAnnotation parses a single line and extracts FR trace annotations.
func extractTraceAnnotation(line string) []string {
	matches := tracesRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil
	}
	var result []string
	for _, p := range strings.Split(matches[1], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func frAnnotationToID(annotation string) string { return strings.ReplaceAll(annotation, "-", "_") }
func frIDToAnnotation(id string) string         { return strings.ReplaceAll(id, "_", "-") }

func validate(frs map[string]FRItem, ucs map[string]UCItem, tsts []TSTItem, features map[string]FeatureSpec, fileTraces map[string][]string, rootDir string) []Violation {
	var violations []Violation
	coveredFRs := make(map[string]bool)
	for _, tst := range tsts {
		for _, link := range tst.FRLinks {
			coveredFRs[link] = true
		}
	}

	// Check uncovered FRs
	for id, fr := range frs {
		if fr.Active && !coveredFRs[id] {
			violations = append(violations, Violation{
				Code:    "UNCOVERED",
				FRID:    id,
				Message: fmt.Sprintf("FR %s is active but has no TST items linked to it", id),
			})
		}
	}

	violations = append(violations, checkRequiredUCsPresent(ucs)...)
	violations = append(violations, checkUCLinks(frs, ucs)...)

	// Check missing annotations and file existence
	violations = append(violations, checkMissingAnnotations(tsts, fileTraces, rootDir)...)

	// Check orphan annotations
	violations = append(violations, checkOrphanAnnotations(frs, fileTraces)...)
	violations = append(violations, checkFeatureScenarioTags(frs, ucs, tsts, features)...)
	violations = append(violations, checkTSTBDDLinks(tsts, features)...)

	return violations
}

func checkFeatureScenarioTags(frs map[string]FRItem, ucs map[string]UCItem, tsts []TSTItem, features map[string]FeatureSpec) []Violation {
	var violations []Violation
	tstIDs := make(map[string]struct{}, len(tsts))
	for _, tst := range tsts {
		tstIDs[normalizeTSTToken(tst.ID)] = struct{}{}
	}
	for _, feature := range features {
		for _, scenario := range feature.Scenarios {
			ucTags := filterTagsByPrefix(scenario.Tags, "@UC-")
			if len(ucTags) != 1 {
				violations = append(violations, Violation{
					Code:    "BDD-UC-TAG-COUNT",
					File:    feature.Path,
					Message: fmt.Sprintf("Scenario %q in %s must include exactly one @UC-* tag", scenario.Name, feature.Path),
				})
			}
			for _, ucTag := range ucTags {
				ucID := tagToDoorstopID(ucTag)
				uc, ok := ucs[ucID]
				if !ok {
					violations = append(violations, Violation{
						Code:    "BDD-UC-TAG-INVALID",
						File:    feature.Path,
						Message: fmt.Sprintf("Scenario %q in %s uses unknown UC tag %s", scenario.Name, feature.Path, ucTag),
					})
					continue
				}
				if uc.BehaviorFamily != "" {
					behaviorTags := filterTagsByPrefix(scenario.Tags, "@behavior-")
					if len(behaviorTags) != 1 {
						violations = append(violations, Violation{
							Code:    "BDD-BEHAVIOR-TAG-COUNT",
							File:    feature.Path,
							Message: fmt.Sprintf("Scenario %q in %s must include exactly one @behavior-* tag for %s", scenario.Name, feature.Path, ucTag),
						})
						continue
					}
					if !behaviorMatchesFamily(behaviorTags[0], uc.BehaviorFamily) {
						violations = append(violations, Violation{
							Code:    "BDD-BEHAVIOR-TAG-INVALID",
							File:    feature.Path,
							Message: fmt.Sprintf("Scenario %q in %s uses behavior tag %s outside family %s", scenario.Name, feature.Path, behaviorTags[0], uc.BehaviorFamily),
						})
					}
				}
			}
			for _, frTag := range filterTagsByPrefix(scenario.Tags, "@FR-") {
				frID := tagToDoorstopID(frTag)
				if _, ok := frs[frID]; !ok {
					violations = append(violations, Violation{
						Code:    "BDD-FR-TAG-INVALID",
						File:    feature.Path,
						Message: fmt.Sprintf("Scenario %q in %s uses unknown FR tag %s", scenario.Name, feature.Path, frTag),
					})
				}
			}
			for _, tstTag := range filterTagsByPrefix(scenario.Tags, "@TST-") {
				if _, ok := tstIDs[normalizeTSTToken(tstTag)]; !ok {
					violations = append(violations, Violation{
						Code:    "BDD-TST-TAG-INVALID",
						File:    feature.Path,
						Message: fmt.Sprintf("Scenario %q in %s uses unknown TST tag %s", scenario.Name, feature.Path, tstTag),
					})
				}
			}
		}
	}
	return violations
}

func checkTSTBDDLinks(tsts []TSTItem, features map[string]FeatureSpec) []Violation {
	var violations []Violation
	for _, tst := range tsts {
		if tst.BDDFeature == "" && tst.BDDScenario == "" && tst.BDDStack == "" && tst.BDDBehavior == "" {
			continue
		}
		if tst.BDDFeature == "" || tst.BDDScenario == "" || tst.BDDStack == "" {
			violations = append(violations, Violation{
				Code:    "BDD-INCOMPLETE",
				TSTID:   tst.ID,
				Message: fmt.Sprintf("TST %s has incomplete BDD metadata", tst.ID),
			})
			continue
		}
		feature, ok := features[tst.BDDFeature]
		if !ok {
			violations = append(violations, Violation{
				Code:    "BDD-FEATURE-NOT-FOUND",
				TSTID:   tst.ID,
				File:    tst.BDDFeature,
				Message: fmt.Sprintf("TST %s points to missing feature %s", tst.ID, tst.BDDFeature),
			})
			continue
		}
		scenario, ok := feature.Scenarios[tst.BDDScenario]
		if !ok {
			violations = append(violations, Violation{
				Code:    "BDD-SCENARIO-NOT-FOUND",
				TSTID:   tst.ID,
				File:    tst.BDDFeature,
				Message: fmt.Sprintf("TST %s points to missing scenario %q in %s", tst.ID, tst.BDDScenario, tst.BDDFeature),
			})
			continue
		}
		if scenario.Stack != tst.BDDStack {
			violations = append(violations, Violation{
				Code:    "BDD-STACK-MISMATCH",
				TSTID:   tst.ID,
				File:    tst.BDDFeature,
				Message: fmt.Sprintf("TST %s declares stack %s but scenario %q uses stack %s", tst.ID, tst.BDDStack, tst.BDDScenario, scenario.Stack),
			})
		}
		if !containsEquivalentTSTTag(scenario.Tags, tst.ID) {
			violations = append(violations, Violation{
				Code:    "BDD-TST-TAG-MISMATCH",
				TSTID:   tst.ID,
				File:    tst.BDDFeature,
				Message: fmt.Sprintf("Scenario %q in %s does not include tag matching %s", tst.BDDScenario, tst.BDDFeature, tst.ID),
			})
		}
		if tst.BDDBehavior != "" && !containsString(scenario.Tags, "@behavior-"+tst.BDDBehavior) {
			violations = append(violations, Violation{
				Code:    "BDD-BEHAVIOR-TAG-MISMATCH",
				TSTID:   tst.ID,
				File:    tst.BDDFeature,
				Message: fmt.Sprintf("Scenario %q in %s does not include tag @behavior-%s", tst.BDDScenario, tst.BDDFeature, tst.BDDBehavior),
			})
		}
		if tst.BDDBehavior != "" && len(tst.FRLinks) > 0 && !sameStringSet(normalizeFeatureFRTags(scenario.Tags), tst.FRLinks) {
			violations = append(violations, Violation{
				Code:    "BDD-FR-LINK-MISMATCH",
				TSTID:   tst.ID,
				File:    tst.BDDFeature,
				Message: fmt.Sprintf("Scenario %q in %s FR tags do not match TST %s FR links", tst.BDDScenario, tst.BDDFeature, tst.ID),
			})
		}
	}
	return violations
}

func behaviorMatchesFamily(tag string, family string) bool {
	behavior := strings.TrimPrefix(tag, "@behavior-")
	prefix := strings.TrimSuffix(family, "*")
	return strings.HasPrefix(behavior, prefix)
}

func normalizeFeatureFRTags(tags []string) []string {
	var frs []string
	for _, tag := range filterTagsByPrefix(tags, "@FR-") {
		frs = append(frs, tagToDoorstopID(tag))
	}
	return frs
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, item := range left {
		counts[item]++
	}
	for _, item := range right {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

func checkRequiredUCsPresent(ucs map[string]UCItem) []Violation {
	var violations []Violation
	for _, requiredID := range requiredUCIDs {
		if _, ok := ucs[requiredID]; !ok {
			violations = append(violations, Violation{
				Code:    "UC-NOT-FOUND",
				Message: fmt.Sprintf("required UC %s is not present in Doorstop", requiredID),
			})
		}
	}
	return violations
}

func checkUCLinks(frs map[string]FRItem, ucs map[string]UCItem) []Violation {
	var violations []Violation
	for ucID, uc := range ucs {
		if !uc.Active {
			continue
		}
		if len(uc.FRLinks) == 0 {
			violations = append(violations, newUCNoFRLinksViolation(ucID))
			continue
		}
		violations = append(violations, findInvalidUCLinks(frs, ucID, uc.FRLinks)...)
	}
	return violations
}

func findInvalidUCLinks(frs map[string]FRItem, ucID string, frLinks []string) []Violation {
	var violations []Violation
	for _, frID := range frLinks {
		if _, ok := frs[frID]; ok {
			continue
		}
		violations = append(violations, Violation{
			Code:    "UC-BAD-FR-LINK",
			FRID:    frID,
			Message: fmt.Sprintf("UC %s links to FR %s but %s is not in Doorstop", ucID, frID, frID),
		})
	}
	return violations
}

func newUCNoFRLinksViolation(ucID string) Violation {
	return Violation{
		Code:    "UC-NO-FR-LINKS",
		Message: fmt.Sprintf("UC %s is active but has no FR links", ucID),
	}
}

// checkMissingAnnotations verifies that test files have required // Traces: annotations.
func checkMissingAnnotations(tsts []TSTItem, fileTraces map[string][]string, rootDir string) []Violation {
	var violations []Violation
	for _, tst := range tsts {
		if tst.BDDFeature != "" && tst.BDDScenario != "" && tst.BDDStack != "" {
			continue
		}
		fullPath := filepath.Join(rootDir, tst.Ref)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			violations = append(violations, Violation{
				Code:    "FILE-NOT-FOUND",
				TSTID:   tst.ID,
				File:    tst.Ref,
				Message: fmt.Sprintf("TST %s ref file %s does not exist", tst.ID, tst.Ref),
			})
			continue
		}

		traces := fileTraces[tst.Ref]
		for _, frLink := range tst.FRLinks {
			expected := frIDToAnnotation(frLink)
			if !containsTrace(traces, expected) {
				violations = append(violations, Violation{
					Code:    "MISSING-ANNOTATION",
					FRID:    frLink,
					TSTID:   tst.ID,
					File:    tst.Ref,
					Message: fmt.Sprintf("TST %s ref file %s lacks annotation '// Traces: %s'", tst.ID, tst.Ref, expected),
				})
			}
		}
	}
	return violations
}

// checkOrphanAnnotations verifies that all // Traces: annotations refer to existing FRs.
func checkOrphanAnnotations(frs map[string]FRItem, fileTraces map[string][]string) []Violation {
	var violations []Violation
	for file, traces := range fileTraces {
		for _, t := range traces {
			frID := frAnnotationToID(t)
			if _, ok := frs[frID]; !ok {
				violations = append(violations, Violation{
					Code:    "ORPHAN",
					FRID:    frID,
					File:    file,
					Message: fmt.Sprintf("File %s has annotation '// Traces: %s' but %s is not in Doorstop", file, t, frID),
				})
			}
		}
	}
	return violations
}

// containsTrace checks if a trace annotation is present in the list.
func containsTrace(traces []string, expected string) bool {
	for _, t := range traces {
		if t == expected {
			return true
		}
	}
	return false
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func containsEquivalentTSTTag(tags []string, tstID string) bool {
	needle := normalizeTSTToken(tstID)
	for _, tag := range tags {
		if strings.HasPrefix(tag, "@TST") && normalizeTSTToken(tag) == needle {
			return true
		}
	}
	return false
}

func filterTagsByPrefix(tags []string, prefix string) []string {
	var filtered []string
	for _, tag := range tags {
		if strings.HasPrefix(tag, prefix) {
			filtered = append(filtered, tag)
		}
	}
	return filtered
}

func tagToDoorstopID(tag string) string {
	return strings.ReplaceAll(strings.TrimPrefix(tag, "@"), "-", "_")
}

func normalizeTSTToken(value string) string {
	value = strings.TrimPrefix(value, "@")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	return value
}

func countActive(frs map[string]FRItem) int {
	c := 0
	for _, fr := range frs {
		if fr.Active {
			c++
		}
	}
	return c
}

func countActiveUCs(ucs map[string]UCItem) int {
	c := 0
	for _, uc := range ucs {
		if uc.Active {
			c++
		}
	}
	return c
}
