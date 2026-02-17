package git

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	MULTI_MERGE_TYPE_BRANCHES = "branches"
	MULTI_MERGE_TYPE_LABELS   = "labels"
)

type MultiMergeDoneError struct{}

// MultiMergeConflictError is returned when a merge conflict occurs during multi-merge
type MultiMergeConflictError struct {
	BranchName string
	Manifest   *MultiMergeManifest
}

func (e *MultiMergeConflictError) Error() string {
	return fmt.Sprintf("merge conflict occurred while merging branch '%s'", e.BranchName)
}

// LocalOnlyBranchesError is returned when branches exist only locally
type LocalOnlyBranchesError struct {
	BranchNames []string
}

func (e *LocalOnlyBranchesError) Error() string {
	return fmt.Sprintf("branches exist only locally: %s", strings.Join(e.BranchNames, ", "))
}

func (r *LocalRepository) MultiMergeNamedBranches(target string, branchNames []string) (*MultiMergeManifest, error) {
	// Make sure we have all changes
	r.Note("Make sure we have all changes")
	fetchOutput, err := r.ExecuteGitCommand("fetch")
	if err != nil {
		return nil, err
	}
	if fetchOutput != "" {
		fmt.Println(fetchOutput)
	}

	// Checkout main branch
	mainBranchName, err := r.MainBranchName()
	if err != nil {
		return nil, err
	}

	// Create todo list
	mainSha, err := r.ExecuteGitCommandQuiet("rev-parse", "--verify", mainBranchName)
	if err != nil {
		return nil, err
	}
	multiMergeManifest := &MultiMergeManifest{
		MainSha:    mainSha,
		Target:     target,
		Type:       MULTI_MERGE_TYPE_BRANCHES,
		References: []MultiMergeReference{},
	}
	multiMergeManifest.MainSha = mainSha

	for _, branchName := range branchNames {
		multiMergeManifest.References = append(multiMergeManifest.References, MultiMergeReference{
			Name: branchName,
		})
	}
	// Delete any existing manifest, to prevent it from interfering with the creation/checkout of the target branch
	multiMergeManifest.Remove()

	// Check if target branch already exists, otherwise create it
	_, err = r.ExecuteGitCommandQuiet("rev-parse", "--verify", target)
	if err == nil {
		r.Note("Checkout target branch")
		r.ExecuteGitCommand("checkout", target)

		r.Note("Make target branch point to %s", mainBranchName)
		_, err = r.ExecuteGitCommand("reset", "--hard", fmt.Sprintf("origin/%s", mainBranchName))
		if err != nil {
			return multiMergeManifest, err
		}
	} else {
		r.Note("Create target branch")
		r.ExecuteGitCommand("checkout", "-b", target)
	}

	// Save current manifest after creation of target branch
	multiMergeManifest.Save()

	for {
		multiMergeManifest, err := r.MultiMergeNamedContinue()
		if err != nil {
			return multiMergeManifest, err
		}

		if multiMergeManifest.IsDone() {
			break
		}
	}

	return multiMergeManifest, nil
}

// Load existing Manifest and redo merges
func (r *LocalRepository) MultiMergeUsingManifest() error {
	// Load & reset manifest
	manifest, err := LoadMultiMergeManifest()
	if err != nil {
		return err
	}
	manifest.Reset()

	// Make sure we have all changes
	r.Note("Make sure we have all changes")
	fetchOutput, err := r.ExecuteGitCommand("fetch")
	if err != nil {
		return err
	}
	if fetchOutput != "" {
		fmt.Println(fetchOutput)
	}

	// Check for local-only branches before making changes
	branchNames := make([]string, len(manifest.References))
	for i, ref := range manifest.References {
		branchNames[i] = ref.Name
	}

	localOnlyBranches, err := r.CheckBranchesHaveRemotes(branchNames)
	if err != nil {
		return err
	}

	if len(localOnlyBranches) > 0 {
		return &LocalOnlyBranchesError{BranchNames: localOnlyBranches}
	}

	// Reset target branch
	r.Note("Checkout target branch")
	r.ExecuteGitCommand("checkout", manifest.Target)

	// Reset target branch to main
	mainBranchName, err := r.MainBranchName()
	if err != nil {
		return err
	}
	r.Note("Make target branch point to %s", mainBranchName)
	_, err = r.ExecuteGitCommand("reset", "--hard", fmt.Sprintf("origin/%s", mainBranchName))
	if err != nil {
		return err
	}

	// Save manifest, which has just been deleted by the hard reset
	manifest.Save()

	// Run merges using "continue"
	for {
		manifest, err := r.MultiMergeNamedContinue()
		if err != nil {
			return err
		}

		if manifest.IsDone() {
			break
		}
	}

	return nil
}

// Multi merge using named labels from PRs
func (r *LocalRepository) MultiMergeNamedLabels(target string, labels []string) error {
	return errors.New("multi merge using PR labels not implemented yet")
}

// Process the rest of the todo list
func (r *LocalRepository) MultiMergeNamedContinue() (*MultiMergeManifest, error) {
	manifest, err := LoadMultiMergeManifest()
	if err != nil {
		return nil, err
	}

	// Find first merge that isn't merged yet
	for i := 0; i < len(manifest.References); i++ {
		reference := &manifest.References[i]
		if reference.Merged {
			// If this is the last branch and is has been merged return with success
			if i == len(manifest.References)-1 {
				r.Note("Last has been merged")
				return manifest, nil
			}
			continue
		}

		// Figure out if this is an ongoing merge, or just the next unmerged branch
		if branchName, err := r.OngoingMergeBranchName(); err == nil && branchName != "" {
			commitMessageBytes, err := os.ReadFile(".git/MERGE_MSG")
			if err != nil {
				return manifest, err
			}
			commitMessageLines := strings.Split(strings.TrimSpace(string(commitMessageBytes)), "\n")
			commitMessage := commitMessageLines[0]

			r.Note("Commit merge")
			commitOutput, err := r.ExecuteGitCommand("commit", "-m", commitMessage)
			if err != nil {
				return manifest, err
			}
			if commitOutput != "" {
				fmt.Println(commitOutput)
			}
			reference.Merged = true
			manifest.Save()
		} else {
			r.Note("Merge branch %s into %s", reference.Name, manifest.Target)

			// Get list of heads matching the branch we want to merge
			heads, err := r.NamedBranches(reference.Name)
			if err != nil {
				r.Warn("Branch %s does not exist, skipping", reference.Name)
				manifest.References = append(manifest.References[:i], manifest.References[i+1:]...)
				i = i - 1
				manifest.Save()
				continue
			}

			// Figure out which branch to merge local or remote (local preferred)
			branchNameToMerge := ""
			if _, exists := heads[fmt.Sprintf("origin/%s", reference.Name)]; exists {
				branchNameToMerge = fmt.Sprintf("origin/%s", reference.Name)
			} else if _, exists := heads[reference.Name]; exists {
				branchNameToMerge = reference.Name
			} else {
				return manifest, fmt.Errorf("unable to find a branch named '%s'", reference.Name)
			}

			mergeOutput, err := r.ExecuteGitCommand("merge", branchNameToMerge)
			if mergeOutput != "" {
				fmt.Println(mergeOutput)
			}
			if err != nil {
				// Check if this is a merge conflict by checking if MERGE_HEAD exists
				if _, statErr := os.Stat(".git/MERGE_HEAD"); statErr == nil {
					return manifest, &MultiMergeConflictError{
						BranchName: branchNameToMerge,
						Manifest:   manifest,
					}
				}
				return manifest, err
			}
			reference.Merged = true
			manifest.Save()
		}
	}

	if manifest.IsDone() {
		r.MultiMergeCommitManifest()
		r.RunHook(PILA_HOOK_MULTI_MERGE_COMPLETE, manifest.Target)
	}

	return manifest, nil
}

func (r *LocalRepository) MultiMergeAbort() error {
	// Load the manifest at the start so we can restore it after reset
	manifest, err := LoadMultiMergeManifest()
	if err != nil {
		return err
	}

	// Abort any running merges
	if branchName, err := r.OngoingMergeBranchName(); err == nil && branchName != "" {
		r.Note("Aborting merge")
		output, err := r.ExecuteGitCommand("merge", "--abort")
		if err != nil {
			panic(err)
		}
		fmt.Println(output)
	}

	// Ensure we're on the target branch
	r.Note("Checkout target branch")
	_, err = r.ExecuteGitCommand("checkout", manifest.Target)
	if err != nil {
		return err
	}

	// Reset target branch to main
	mainBranchName, err := r.MainBranchName()
	if err != nil {
		panic(err)
	}
	output, err := r.ExecuteGitCommand("reset", "--hard", fmt.Sprintf("origin/%s", mainBranchName))
	fmt.Println(output)
	if err != nil {
		return err
	}

	// Reset manifest state (mark all branches as unmerged) and save it
	manifest.Reset()
	return manifest.Save()
}

type MultiMergeTestBranchResult struct {
	Name             string   `json:"name"`
	Status           string   `json:"status"`                      // "clean", "conflict", "missing"
	MergeType        string   `json:"merge_type,omitempty"`        // "sequential" or "main-only"
	ConflictingFiles []string `json:"conflicting_files,omitempty"` // only when Status == "conflict"
}

type MultiMergeTestResult struct {
	OK            bool                          `json:"ok"`
	Error         string                        `json:"error,omitempty"`
	BranchResults []MultiMergeTestBranchResult `json:"branches"`
}

func (r *LocalRepository) MultiMergeTest() (*MultiMergeTestResult, error) {
	manifest, err := LoadMultiMergeManifest()
	if err != nil {
		return nil, err
	}

	// Fetch latest changes
	r.Note("Make sure we have all changes")
	fetchOutput, err := r.ExecuteGitCommand("fetch")
	if err != nil {
		return nil, err
	}
	if fetchOutput != "" {
		fmt.Println(fetchOutput)
	}

	// Save current branch name
	originalBranch, err := r.ExecuteGitCommandQuiet("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}

	mainBranchName, err := r.MainBranchName()
	if err != nil {
		return nil, err
	}

	// Checkout detached HEAD at origin/<main>
	r.Note("Checkout detached HEAD at origin/%s", mainBranchName)
	_, err = r.ExecuteGitCommand("checkout", "--detach", fmt.Sprintf("origin/%s", mainBranchName))
	if err != nil {
		// Restore original branch before returning
		r.ExecuteGitCommandQuiet("checkout", originalBranch)
		return nil, err
	}

	result := &MultiMergeTestResult{
		OK:            true,
		BranchResults: []MultiMergeTestBranchResult{},
	}
	sequential := true

	for _, reference := range manifest.References {
		// Resolve branch name (prefer origin/<branch> over local)
		heads, err := r.NamedBranches(reference.Name)
		if err != nil {
			result.BranchResults = append(result.BranchResults, MultiMergeTestBranchResult{
				Name:   reference.Name,
				Status: "missing",
			})
			continue
		}

		branchNameToMerge := ""
		if _, exists := heads[fmt.Sprintf("origin/%s", reference.Name)]; exists {
			branchNameToMerge = fmt.Sprintf("origin/%s", reference.Name)
		} else if _, exists := heads[reference.Name]; exists {
			branchNameToMerge = reference.Name
		} else {
			result.BranchResults = append(result.BranchResults, MultiMergeTestBranchResult{
				Name:   reference.Name,
				Status: "missing",
			})
			continue
		}

		// If a prior branch conflicted, reset to main for a clean "branch vs main" test
		if !sequential {
			r.ExecuteGitCommandQuiet("reset", "--hard", fmt.Sprintf("origin/%s", mainBranchName))
		}

		mergeType := "sequential"
		if !sequential {
			mergeType = "main-only"
		}

		r.Note("Test merge %s (%s)", reference.Name, mergeType)
		_, mergeErr := r.ExecuteGitCommand("merge", "--no-ff", "--no-commit", branchNameToMerge)

		if mergeErr == nil {
			// Clean merge
			if sequential {
				// Commit to advance the base for subsequent branches
				r.ExecuteGitCommandQuiet("commit", "-m", fmt.Sprintf("test merge %s", reference.Name))
			} else {
				r.ExecuteGitCommandQuiet("merge", "--abort")
			}
			result.BranchResults = append(result.BranchResults, MultiMergeTestBranchResult{
				Name:      reference.Name,
				Status:    "clean",
				MergeType: mergeType,
			})
		} else {
			// Check if this is actually a merge conflict
			conflictingFiles := []string{}
			filesOutput, filesErr := r.ExecuteGitCommandQuiet("diff", "--name-only", "--diff-filter=U")
			if filesErr == nil && filesOutput != "" {
				conflictingFiles = strings.Split(filesOutput, "\n")
			}

			r.ExecuteGitCommandQuiet("merge", "--abort")
			sequential = false
			result.OK = false
			result.BranchResults = append(result.BranchResults, MultiMergeTestBranchResult{
				Name:             reference.Name,
				Status:           "conflict",
				MergeType:        mergeType,
				ConflictingFiles: conflictingFiles,
			})
		}
	}

	// Restore original state
	r.Note("Restore original branch")
	r.ExecuteGitCommand("checkout", originalBranch)

	return result, nil
}

func (r *LocalRepository) MultiMergeCommitManifest() error {
	r.Note("Adding manifest to git")
	output, err := r.ExecuteGitCommand("add", ".pila_multi_merge.yaml")
	if err != nil {
		return err
	}
	if output != "" {
		fmt.Println(output)
	}

	r.Note("Committing manifest to git")
	output, err = r.ExecuteGitCommand("commit", "-m", "chore: Add Pila multi merge manifest")
	if err != nil {
		return err
	}
	fmt.Println(output)

	return nil
}
