package git

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Compile-time interface checks
var _ error = (*MultiMergeConflictError)(nil)
var _ error = (*LocalOnlyBranchesError)(nil)

func TestMultiMergeConflictError_Error(t *testing.T) {
	err := &MultiMergeConflictError{BranchName: "feature-x"}
	got := err.Error()
	want := "merge conflict occurred while merging branch 'feature-x'"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestLocalOnlyBranchesError_Error_SingleBranch(t *testing.T) {
	err := &LocalOnlyBranchesError{BranchNames: []string{"my-branch"}}
	got := err.Error()
	want := "branches exist only locally: my-branch"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestLocalOnlyBranchesError_Error_MultipleBranches(t *testing.T) {
	err := &LocalOnlyBranchesError{BranchNames: []string{"branch-a", "branch-b", "branch-c"}}
	got := err.Error()
	if !strings.Contains(got, "branch-a") || !strings.Contains(got, "branch-b") || !strings.Contains(got, "branch-c") {
		t.Errorf("Error() = %q, expected all branch names to be present", got)
	}
	want := "branches exist only locally: branch-a, branch-b, branch-c"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestMultiMergeTestResult_JSONSerialization_ErrorOmittedWhenEmpty(t *testing.T) {
	result := MultiMergeTestResult{
		OK:            true,
		BranchResults: []MultiMergeTestBranchResult{},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// The "error" key should be absent from JSON when Error is empty
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, exists := raw["error"]; exists {
		t.Errorf("expected \"error\" key to be absent from JSON, got: %s", string(data))
	}
}

func TestMultiMergeTestResult_JSONSerialization_ErrorPresentWhenSet(t *testing.T) {
	result := MultiMergeTestResult{
		OK:            false,
		Error:         "something went wrong",
		BranchResults: []MultiMergeTestBranchResult{},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// The "error" key should be present with the correct value
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	errorJSON, exists := raw["error"]
	if !exists {
		t.Fatalf("expected \"error\" key to be present in JSON, got: %s", string(data))
	}

	var errorValue string
	if err := json.Unmarshal(errorJSON, &errorValue); err != nil {
		t.Fatalf("json.Unmarshal(error) error = %v", err)
	}
	if errorValue != "something went wrong" {
		t.Errorf("error = %q, want %q", errorValue, "something went wrong")
	}
}

func TestMultiMergeTestBranchResult_JSONSerialization_ErrorOmittedWhenEmpty(t *testing.T) {
	br := MultiMergeTestBranchResult{
		Name:      "feature-a",
		Status:    "clean",
		MergeType: "sequential",
	}

	data, err := json.Marshal(br)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, exists := raw["error"]; exists {
		t.Errorf("expected \"error\" key to be absent from JSON, got: %s", string(data))
	}
}

func TestMultiMergeTestBranchResult_JSONSerialization_ErrorPresentWhenSet(t *testing.T) {
	br := MultiMergeTestBranchResult{
		Name:      "feature-a",
		Status:    "error",
		MergeType: "sequential",
		Error:     "git merge failed: uncommitted changes",
	}

	data, err := json.Marshal(br)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	errorJSON, exists := raw["error"]
	if !exists {
		t.Fatalf("expected \"error\" key to be present in JSON, got: %s", string(data))
	}

	var errorValue string
	if err := json.Unmarshal(errorJSON, &errorValue); err != nil {
		t.Fatalf("json.Unmarshal(error) error = %v", err)
	}
	if errorValue != "git merge failed: uncommitted changes" {
		t.Errorf("error = %q, want %q", errorValue, "git merge failed: uncommitted changes")
	}
}

// runGit executes a git command in the given directory and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// setupTestRepo creates a bare origin repo and a cloned working copy with test branches.
// Branches created:
//   - feature-clean: adds clean.txt with "clean content"
//   - feature-conflict: adds clean.txt with "conflict content" (conflicts with feature-clean)
//   - feature-clean2: adds clean2.txt (no conflict with anything)
//
// Returns the workdir path. Changes cwd to workdir; cleanup restores original cwd.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	parentDir := t.TempDir()
	originDir := filepath.Join(parentDir, "origin.git")
	workDir := filepath.Join(parentDir, "workdir")

	// Create bare origin
	runGit(t, parentDir, "init", "--bare", originDir)

	// Clone it
	runGit(t, parentDir, "clone", originDir, workDir)

	// Configure user in workdir
	runGit(t, workDir, "config", "user.email", "test@test.com")
	runGit(t, workDir, "config", "user.name", "Test User")

	// Create initial commit on main
	initialFile := filepath.Join(workDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("writing README.md: %v", err)
	}
	runGit(t, workDir, "add", "README.md")
	runGit(t, workDir, "commit", "-m", "initial commit")
	runGit(t, workDir, "push", "origin", "main")
	runGit(t, workDir, "remote", "set-head", "origin", "main")

	// Create feature-clean branch: adds clean.txt
	runGit(t, workDir, "checkout", "-b", "feature-clean", "main")
	if err := os.WriteFile(filepath.Join(workDir, "clean.txt"), []byte("clean content\n"), 0644); err != nil {
		t.Fatalf("writing clean.txt: %v", err)
	}
	runGit(t, workDir, "add", "clean.txt")
	runGit(t, workDir, "commit", "-m", "add clean.txt")
	runGit(t, workDir, "push", "origin", "feature-clean")

	// Create feature-conflict branch: adds clean.txt with different content (conflicts with feature-clean)
	runGit(t, workDir, "checkout", "main")
	runGit(t, workDir, "checkout", "-b", "feature-conflict")
	if err := os.WriteFile(filepath.Join(workDir, "clean.txt"), []byte("conflict content\n"), 0644); err != nil {
		t.Fatalf("writing clean.txt: %v", err)
	}
	runGit(t, workDir, "add", "clean.txt")
	runGit(t, workDir, "commit", "-m", "add conflicting clean.txt")
	runGit(t, workDir, "push", "origin", "feature-conflict")

	// Create feature-clean2 branch: adds clean2.txt (different file, no conflict)
	runGit(t, workDir, "checkout", "main")
	runGit(t, workDir, "checkout", "-b", "feature-clean2")
	if err := os.WriteFile(filepath.Join(workDir, "clean2.txt"), []byte("clean2 content\n"), 0644); err != nil {
		t.Fatalf("writing clean2.txt: %v", err)
	}
	runGit(t, workDir, "add", "clean2.txt")
	runGit(t, workDir, "commit", "-m", "add clean2.txt")
	runGit(t, workDir, "push", "origin", "feature-clean2")

	// Return to main
	runGit(t, workDir, "checkout", "main")

	// Change to workdir
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir to workdir: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(originalDir)
	})

	return workDir
}

// writeTestManifest writes a .pila_multi_merge.yaml with the given branch names.
func writeTestManifest(t *testing.T, workDir string, branchNames []string) {
	t.Helper()
	lines := []string{
		"main_sha: \"\"",
		"target: \"\"",
		"type: branches",
		"references:",
	}
	for _, name := range branchNames {
		lines = append(lines, fmt.Sprintf("    - name: %s", name))
		lines = append(lines, "      merged: false")
	}
	content := strings.Join(lines, "\n") + "\n"
	path := filepath.Join(workDir, MULTI_MERGE_MANIFEST_FILENAME)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
}

// suppressStdout redirects os.Stdout to /dev/null for the duration of the test,
// keeping test output pristine from git command traces.
func suppressStdout(t *testing.T) {
	t.Helper()
	origStdout := os.Stdout
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("opening /dev/null: %v", err)
	}
	os.Stdout = devNull
	t.Cleanup(func() {
		os.Stdout = origStdout
		devNull.Close()
	})
}

func TestMultiMergeTest_AllClean(t *testing.T) {
	workDir := setupTestRepo(t)
	suppressStdout(t)

	writeTestManifest(t, workDir, []string{"feature-clean", "feature-clean2"})

	repo, err := GetLocalRepository()
	if err != nil {
		t.Fatalf("GetLocalRepository() error = %v", err)
	}

	result, err := repo.MultiMergeTest()
	if err != nil {
		t.Fatalf("MultiMergeTest() error = %v", err)
	}

	if !result.OK {
		t.Errorf("result.OK = false, want true")
	}
	if len(result.BranchResults) != 2 {
		t.Fatalf("got %d branch results, want 2", len(result.BranchResults))
	}

	for i, br := range result.BranchResults {
		if br.Status != "clean" {
			t.Errorf("branch %d (%s): status = %q, want %q", i, br.Name, br.Status, "clean")
		}
		if br.MergeType != "sequential" {
			t.Errorf("branch %d (%s): merge_type = %q, want %q", i, br.Name, br.MergeType, "sequential")
		}
	}

	// Verify original branch was restored
	currentBranch := runGit(t, workDir, "rev-parse", "--abbrev-ref", "HEAD")
	if currentBranch != "main" {
		t.Errorf("current branch = %q, want %q", currentBranch, "main")
	}
}

func TestMultiMergeTest_WithConflict(t *testing.T) {
	workDir := setupTestRepo(t)
	suppressStdout(t)

	writeTestManifest(t, workDir, []string{"feature-clean", "feature-conflict"})

	repo, err := GetLocalRepository()
	if err != nil {
		t.Fatalf("GetLocalRepository() error = %v", err)
	}

	result, err := repo.MultiMergeTest()
	if err != nil {
		t.Fatalf("MultiMergeTest() error = %v", err)
	}

	if result.OK {
		t.Errorf("result.OK = true, want false")
	}
	if len(result.BranchResults) != 2 {
		t.Fatalf("got %d branch results, want 2", len(result.BranchResults))
	}

	// First branch should be clean
	if result.BranchResults[0].Status != "clean" {
		t.Errorf("branch 0 (%s): status = %q, want %q", result.BranchResults[0].Name, result.BranchResults[0].Status, "clean")
	}
	if result.BranchResults[0].MergeType != "sequential" {
		t.Errorf("branch 0 (%s): merge_type = %q, want %q", result.BranchResults[0].Name, result.BranchResults[0].MergeType, "sequential")
	}

	// Second branch should conflict
	if result.BranchResults[1].Status != "conflict" {
		t.Errorf("branch 1 (%s): status = %q, want %q", result.BranchResults[1].Name, result.BranchResults[1].Status, "conflict")
	}
	if result.BranchResults[1].MergeType != "sequential" {
		t.Errorf("branch 1 (%s): merge_type = %q, want %q", result.BranchResults[1].Name, result.BranchResults[1].MergeType, "sequential")
	}

	foundCleanTxt := false
	for _, f := range result.BranchResults[1].ConflictingFiles {
		if f == "clean.txt" {
			foundCleanTxt = true
		}
	}
	if !foundCleanTxt {
		t.Errorf("branch 1 conflicting files = %v, expected to contain 'clean.txt'", result.BranchResults[1].ConflictingFiles)
	}
}

func TestMultiMergeTest_ContinuesOnStackAfterConflict(t *testing.T) {
	workDir := setupTestRepo(t)
	suppressStdout(t)

	writeTestManifest(t, workDir, []string{"feature-clean", "feature-conflict", "feature-clean2"})

	repo, err := GetLocalRepository()
	if err != nil {
		t.Fatalf("GetLocalRepository() error = %v", err)
	}

	result, err := repo.MultiMergeTest()
	if err != nil {
		t.Fatalf("MultiMergeTest() error = %v", err)
	}

	if result.OK {
		t.Errorf("result.OK = true, want false")
	}
	if len(result.BranchResults) != 3 {
		t.Fatalf("got %d branch results, want 3", len(result.BranchResults))
	}

	// Branch 0: clean, sequential
	if result.BranchResults[0].Status != "clean" {
		t.Errorf("branch 0: status = %q, want %q", result.BranchResults[0].Status, "clean")
	}
	if result.BranchResults[0].MergeType != "sequential" {
		t.Errorf("branch 0: merge_type = %q, want %q", result.BranchResults[0].MergeType, "sequential")
	}

	// Branch 1: conflict, sequential
	if result.BranchResults[1].Status != "conflict" {
		t.Errorf("branch 1: status = %q, want %q", result.BranchResults[1].Status, "conflict")
	}
	if result.BranchResults[1].MergeType != "sequential" {
		t.Errorf("branch 1: merge_type = %q, want %q", result.BranchResults[1].MergeType, "sequential")
	}

	// Branch 2: clean, still merged onto the stack after the conflict
	if result.BranchResults[2].Status != "clean" {
		t.Errorf("branch 2: status = %q, want %q", result.BranchResults[2].Status, "clean")
	}
	if result.BranchResults[2].MergeType != "sequential" {
		t.Errorf("branch 2: merge_type = %q, want %q", result.BranchResults[2].MergeType, "sequential")
	}
}

func TestMultiMergeTest_DetectsConflictWithStackAfterEarlierConflict(t *testing.T) {
	workDir := setupTestRepo(t)
	suppressStdout(t)

	// feature-conflict2 merges cleanly into main on its own, but conflicts with
	// feature-clean which is already part of the stack when it is tested.
	runGit(t, workDir, "checkout", "-b", "feature-conflict2", "main")
	if err := os.WriteFile(filepath.Join(workDir, "clean.txt"), []byte("another conflicting content\n"), 0644); err != nil {
		t.Fatalf("writing clean.txt: %v", err)
	}
	runGit(t, workDir, "add", "clean.txt")
	runGit(t, workDir, "commit", "-m", "add second conflicting clean.txt")
	runGit(t, workDir, "push", "origin", "feature-conflict2")
	runGit(t, workDir, "checkout", "main")

	writeTestManifest(t, workDir, []string{"feature-clean", "feature-conflict", "feature-conflict2"})

	repo, err := GetLocalRepository()
	if err != nil {
		t.Fatalf("GetLocalRepository() error = %v", err)
	}

	result, err := repo.MultiMergeTest()
	if err != nil {
		t.Fatalf("MultiMergeTest() error = %v", err)
	}

	if result.OK {
		t.Errorf("result.OK = true, want false")
	}
	if len(result.BranchResults) != 3 {
		t.Fatalf("got %d branch results, want 3", len(result.BranchResults))
	}

	// Branch 2 conflicts with the stack (feature-clean), even though it merges cleanly into main
	if result.BranchResults[2].Status != "conflict" {
		t.Errorf("branch 2: status = %q, want %q", result.BranchResults[2].Status, "conflict")
	}
	if result.BranchResults[2].MergeType != "sequential" {
		t.Errorf("branch 2: merge_type = %q, want %q", result.BranchResults[2].MergeType, "sequential")
	}

	foundCleanTxt := false
	for _, f := range result.BranchResults[2].ConflictingFiles {
		if f == "clean.txt" {
			foundCleanTxt = true
		}
	}
	if !foundCleanTxt {
		t.Errorf("branch 2 conflicting files = %v, expected to contain 'clean.txt'", result.BranchResults[2].ConflictingFiles)
	}

	// Verify original branch was restored
	currentBranch := runGit(t, workDir, "rev-parse", "--abbrev-ref", "HEAD")
	if currentBranch != "main" {
		t.Errorf("current branch = %q, want %q", currentBranch, "main")
	}
}

func TestMultiMergeTest_MissingBranch(t *testing.T) {
	workDir := setupTestRepo(t)
	suppressStdout(t)

	writeTestManifest(t, workDir, []string{"feature-clean", "nonexistent-branch"})

	repo, err := GetLocalRepository()
	if err != nil {
		t.Fatalf("GetLocalRepository() error = %v", err)
	}

	result, err := repo.MultiMergeTest()
	if err != nil {
		t.Fatalf("MultiMergeTest() error = %v", err)
	}

	if len(result.BranchResults) != 2 {
		t.Fatalf("got %d branch results, want 2", len(result.BranchResults))
	}

	// Missing branches do not fail the result
	if !result.OK {
		t.Errorf("result.OK = false, want true (missing branches don't fail)")
	}

	if result.BranchResults[1].Status != "missing" {
		t.Errorf("branch 1: status = %q, want %q", result.BranchResults[1].Status, "missing")
	}
}

func TestMultiMergeTest_NoManifest(t *testing.T) {
	workDir := setupTestRepo(t)
	suppressStdout(t)

	// Ensure no manifest file exists
	os.Remove(filepath.Join(workDir, MULTI_MERGE_MANIFEST_FILENAME))

	repo, err := GetLocalRepository()
	if err != nil {
		t.Fatalf("GetLocalRepository() error = %v", err)
	}

	result, err := repo.MultiMergeTest()
	if err == nil {
		t.Fatal("MultiMergeTest() expected error for missing manifest, got nil")
	}
	if result != nil {
		t.Errorf("result = %v, want nil when manifest is missing", result)
	}
}
