package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"go.olrik.dev/pila/internal/git"
)

func TestWriteResultToFile_SuccessResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "result.json")

	result := &git.MultiMergeTestResult{
		OK: true,
		BranchResults: []git.MultiMergeTestBranchResult{
			{Name: "feature-a", Status: "clean", MergeType: "sequential"},
		},
	}

	err := writeResultToFile(path, result)
	if err != nil {
		t.Fatalf("writeResultToFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got git.MultiMergeTestResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !got.OK {
		t.Errorf("got OK = false, want true")
	}
	if got.Error != "" {
		t.Errorf("got Error = %q, want empty", got.Error)
	}
	if len(got.BranchResults) != 1 {
		t.Fatalf("got %d branches, want 1", len(got.BranchResults))
	}
	if got.BranchResults[0].Name != "feature-a" {
		t.Errorf("got branch name = %q, want %q", got.BranchResults[0].Name, "feature-a")
	}
}

func TestWriteResultToFile_ErrorResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "result.json")

	result := &git.MultiMergeTestResult{
		OK:            false,
		Error:         "no manifest found",
		BranchResults: []git.MultiMergeTestBranchResult{},
	}

	err := writeResultToFile(path, result)
	if err != nil {
		t.Fatalf("writeResultToFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var got git.MultiMergeTestResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.OK {
		t.Errorf("got OK = true, want false")
	}
	if got.Error != "no manifest found" {
		t.Errorf("got Error = %q, want %q", got.Error, "no manifest found")
	}
}

func TestWriteResultToFile_InvalidPath(t *testing.T) {
	result := &git.MultiMergeTestResult{
		OK:            true,
		BranchResults: []git.MultiMergeTestBranchResult{},
	}

	err := writeResultToFile("/nonexistent/dir/result.json", result)
	if err == nil {
		t.Fatal("writeResultToFile() expected error for invalid path, got nil")
	}
}

func TestMultiMergeTestCommand_HasOutputFlag(t *testing.T) {
	cmd := NewMultiMergeTestCommand()

	flag := cmd.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("expected --output flag to exist")
	}
	if flag.Shorthand != "O" {
		t.Errorf("expected shorthand -O, got -%s", flag.Shorthand)
	}
}

func TestFilterDuplicateBranches_NoDuplicates(t *testing.T) {
	existing := []string{"a", "b"}
	newBranches := []string{"c", "d"}

	filtered, duplicates := filterDuplicateBranches(existing, newBranches)

	if len(filtered) != 2 || filtered[0] != "c" || filtered[1] != "d" {
		t.Errorf("filtered = %v, want [c d]", filtered)
	}
	if len(duplicates) != 0 {
		t.Errorf("duplicates = %v, want []", duplicates)
	}
}

func TestFilterDuplicateBranches_AllDuplicates(t *testing.T) {
	existing := []string{"a", "b"}
	newBranches := []string{"a", "b"}

	filtered, duplicates := filterDuplicateBranches(existing, newBranches)

	if len(filtered) != 0 {
		t.Errorf("filtered = %v, want []", filtered)
	}
	if len(duplicates) != 2 || duplicates[0] != "a" || duplicates[1] != "b" {
		t.Errorf("duplicates = %v, want [a b]", duplicates)
	}
}

func TestFilterDuplicateBranches_MixedDuplicates(t *testing.T) {
	existing := []string{"a", "b"}
	newBranches := []string{"b", "c", "a", "d"}

	filtered, duplicates := filterDuplicateBranches(existing, newBranches)

	if len(filtered) != 2 || filtered[0] != "c" || filtered[1] != "d" {
		t.Errorf("filtered = %v, want [c d]", filtered)
	}
	if len(duplicates) != 2 || duplicates[0] != "b" || duplicates[1] != "a" {
		t.Errorf("duplicates = %v, want [b a]", duplicates)
	}
}

func TestFilterDuplicateBranches_EmptyExisting(t *testing.T) {
	existing := []string{}
	newBranches := []string{"a", "b"}

	filtered, duplicates := filterDuplicateBranches(existing, newBranches)

	if len(filtered) != 2 || filtered[0] != "a" || filtered[1] != "b" {
		t.Errorf("filtered = %v, want [a b]", filtered)
	}
	if len(duplicates) != 0 {
		t.Errorf("duplicates = %v, want []", duplicates)
	}
}

func TestFilterDuplicateBranches_EmptyNew(t *testing.T) {
	existing := []string{"a", "b"}
	newBranches := []string{}

	filtered, duplicates := filterDuplicateBranches(existing, newBranches)

	if len(filtered) != 0 {
		t.Errorf("filtered = %v, want []", filtered)
	}
	if len(duplicates) != 0 {
		t.Errorf("duplicates = %v, want []", duplicates)
	}
}

func TestFilterDuplicateBranches_DuplicatesWithinNewBranches(t *testing.T) {
	existing := []string{}
	newBranches := []string{"a", "a"}

	filtered, duplicates := filterDuplicateBranches(existing, newBranches)

	if len(filtered) != 1 || filtered[0] != "a" {
		t.Errorf("filtered = %v, want [a]", filtered)
	}
	if len(duplicates) != 1 || duplicates[0] != "a" {
		t.Errorf("duplicates = %v, want [a]", duplicates)
	}
}

func TestMultiMergeTestCommand_NoJsonFlag(t *testing.T) {
	cmd := NewMultiMergeTestCommand()

	flag := cmd.Flags().Lookup("json")
	if flag != nil {
		t.Fatal("expected --json flag to not exist")
	}
}

func TestPrintTestResult_AllClean(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	result := &git.MultiMergeTestResult{
		OK: true,
		BranchResults: []git.MultiMergeTestBranchResult{
			{Name: "feature-a", Status: "clean", MergeType: "sequential"},
			{Name: "feature-b", Status: "clean", MergeType: "sequential"},
		},
	}

	printTestResult(&buf, result)
	output := buf.String()

	if !strings.Contains(output, "feature-a") {
		t.Errorf("output missing branch name 'feature-a', got:\n%s", output)
	}
	if !strings.Contains(output, "feature-b") {
		t.Errorf("output missing branch name 'feature-b', got:\n%s", output)
	}
	if !strings.Contains(output, "All branches merge cleanly.") {
		t.Errorf("output missing success message, got:\n%s", output)
	}
}

func TestPrintTestResult_Conflict(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	result := &git.MultiMergeTestResult{
		OK: false,
		BranchResults: []git.MultiMergeTestBranchResult{
			{Name: "feature-conflict", Status: "conflict", MergeType: "sequential", ConflictingFiles: []string{"file1.txt", "file2.txt"}},
		},
	}

	printTestResult(&buf, result)
	output := buf.String()

	if !strings.Contains(output, "feature-conflict") {
		t.Errorf("output missing branch name 'feature-conflict', got:\n%s", output)
	}
	if !strings.Contains(output, "file1.txt") {
		t.Errorf("output missing conflicting file 'file1.txt', got:\n%s", output)
	}
	if !strings.Contains(output, "file2.txt") {
		t.Errorf("output missing conflicting file 'file2.txt', got:\n%s", output)
	}
	if !strings.Contains(output, "Some branches have conflicts.") {
		t.Errorf("output missing conflict message, got:\n%s", output)
	}
}

func TestPrintTestResult_Missing(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	result := &git.MultiMergeTestResult{
		OK: true,
		BranchResults: []git.MultiMergeTestBranchResult{
			{Name: "gone-branch", Status: "missing"},
		},
	}

	printTestResult(&buf, result)
	output := buf.String()

	if !strings.Contains(output, "gone-branch") {
		t.Errorf("output missing branch name 'gone-branch', got:\n%s", output)
	}
	if !strings.Contains(output, "(missing)") {
		t.Errorf("output missing '(missing)' indicator, got:\n%s", output)
	}
}

func TestPrintTestResult_Error(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	result := &git.MultiMergeTestResult{
		OK: false,
		BranchResults: []git.MultiMergeTestBranchResult{
			{Name: "bad-branch", Status: "error", MergeType: "sequential", Error: "unexpected git failure"},
		},
	}

	printTestResult(&buf, result)
	output := buf.String()

	if !strings.Contains(output, "bad-branch") {
		t.Errorf("output missing branch name 'bad-branch', got:\n%s", output)
	}
	if !strings.Contains(output, "(error)") {
		t.Errorf("output missing '(error)' indicator, got:\n%s", output)
	}
	if !strings.Contains(output, "unexpected git failure") {
		t.Errorf("output missing error message, got:\n%s", output)
	}
}

func TestPrintTestResult_MixedStatuses(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer
	result := &git.MultiMergeTestResult{
		OK: false,
		BranchResults: []git.MultiMergeTestBranchResult{
			{Name: "clean-branch", Status: "clean", MergeType: "sequential"},
			{Name: "conflict-branch", Status: "conflict", MergeType: "sequential", ConflictingFiles: []string{"shared.go"}},
			{Name: "missing-branch", Status: "missing"},
			{Name: "error-branch", Status: "error", MergeType: "sequential", Error: "merge failed"},
		},
	}

	printTestResult(&buf, result)
	output := buf.String()

	for _, want := range []string{"clean-branch", "conflict-branch", "missing-branch", "error-branch", "shared.go", "(missing)", "(error)", "merge failed", "Some branches have conflicts."} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q, got:\n%s", want, output)
		}
	}
}
