package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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

func TestMultiMergeTestCommand_NoJsonFlag(t *testing.T) {
	cmd := NewMultiMergeTestCommand()

	flag := cmd.Flags().Lookup("json")
	if flag != nil {
		t.Fatal("expected --json flag to not exist")
	}
}
