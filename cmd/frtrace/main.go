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

type TSTItem struct {
	ID      string
	Ref     string
	FRLinks []string
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
	flagReqs        = "reqs"
	defaultReqsDir  = "./reqs"
	extYAML         = ".yml"
	doorstopYAML    = ".doorstop.yml"
)

func main() {
	reqsDir := flag.String(flagReqs, defaultReqsDir, "Path to Doorstop requirements directory")
	rootDir := flag.String("root", ".", "Project root directory")
	flag.Parse()

	frs, err := loadDoorstopFRs(filepath.Join(*reqsDir, "FR"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading FRs: %v\n", err)
		os.Exit(1)
	}
	tsts, err := loadDoorstopTSTs(filepath.Join(*reqsDir, "TST"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR loading TSTs: %v\n", err)
		os.Exit(1)
	}

	fileTraces := make(map[string][]string)
	for _, tst := range tsts {
		fullPath := filepath.Join(*rootDir, tst.Ref)
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

	violations := validate(frs, tsts, fileTraces, *rootDir)
	fmt.Printf("=== FR Traceability Report ===\n")
	fmt.Printf("FRs loaded: %d (active: %d)\n", len(frs), countActive(frs))
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

func loadDoorstopFRs(dir string) (map[string]FRItem, error) {
	frs := make(map[string]FRItem)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading FR directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), extYAML) || entry.Name() == doorstopYAML {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), readErr)
		}
		var fr FRItem
		if parseErr := yaml.Unmarshal(data, &fr); parseErr != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), parseErr)
		}
		id := strings.TrimSuffix(entry.Name(), extYAML)
		frs[id] = fr
	}
	return frs, nil
}

func loadDoorstopTSTs(dir string) ([]TSTItem, error) {
	var tsts []TSTItem
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading TST directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), extYAML) || entry.Name() == doorstopYAML {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), readErr)
		}
		var raw tstYAML
		if parseErr := yaml.Unmarshal(data, &raw); parseErr != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), parseErr)
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
			result = append(result, v)
		case map[string]interface{}:
			for key := range v {
				result = append(result, key)
			}
		}
	}
	return result
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

func validate(frs map[string]FRItem, tsts []TSTItem, fileTraces map[string][]string, rootDir string) []Violation {
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

	// Check missing annotations and file existence
	violations = append(violations, checkMissingAnnotations(tsts, fileTraces, rootDir)...)

	// Check orphan annotations
	violations = append(violations, checkOrphanAnnotations(frs, fileTraces)...)

	return violations
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
