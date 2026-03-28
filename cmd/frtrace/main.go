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
	Active         bool     `yaml:"active"`
	BehaviorFamily string   `yaml:"behavior_family"`
	FRLinks        []string
}

type TSTItem struct {
	ID      string
	Ref     string
	FRLinks []string
}

type ucYAML struct {
	Active bool        `yaml:"active"`
	Links  interface{} `yaml:"links"`
}

type tstYAML struct {
	Active bool        `yaml:"active"`
	Ref    string      `yaml:"ref"`
	Links  interface{} `yaml:"links"`
}

type Violation struct {
	Code    string
	FRID    string
	TSTID   string
	File    string
	Message string
}

const (
	flagReqs       = "reqs"
	defaultReqsDir = "./reqs"
	extYAML        = ".yml"
	doorstopYAML   = ".doorstop.yml"
	errReadFileFmt = "reading %s: %w"
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

	fileTraces := buildFileTraces(tsts, *rootDir)
	violations := validate(frs, ucs, tsts, fileTraces, *rootDir)
	printReport(frs, ucs, tsts, fileTraces, violations)
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

func printReport(frs map[string]FRItem, ucs map[string]UCItem, tsts []TSTItem, fileTraces map[string][]string, violations []Violation) {
	fmt.Printf("=== FR Traceability Report ===\n")
	fmt.Printf("FRs loaded: %d (active: %d)\n", len(frs), countActive(frs))
	fmt.Printf("UCs loaded: %d (active: %d)\n", len(ucs), countActiveUCs(ucs))
	fmt.Printf("TST items loaded: %d\n", len(tsts))
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
			Active:  raw.Active,
			FRLinks: extractFRLinks(raw.Links),
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
		tsts = append(tsts, TSTItem{ID: id, Ref: raw.Ref, FRLinks: extractFRLinks(raw.Links)})
	}
	return tsts, nil
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

func validate(frs map[string]FRItem, ucs map[string]UCItem, tsts []TSTItem, fileTraces map[string][]string, rootDir string) []Violation {
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

	return violations
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
