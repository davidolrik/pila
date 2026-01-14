package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func Heads() (map[string]string, error) {
	heads := make(map[string]string)

	cmd := exec.Command("git", "show-ref", "--branches")
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return heads, err
	}

	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, " ")
		parts[1] = strings.Replace(parts[1], "refs/heads/", "", 1)
		heads[parts[1]] = parts[0]
	}

	return heads, nil
}

func (r *LocalRepository) AllBranchNames() ([]string, error) {
	branches := []string{}

	output, err := r.ExecuteGitCommandQuiet("show-ref")
	if err != nil {
		return branches, nil
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.Split(line, " ")
		if strings.HasPrefix(parts[1], "refs/heads/") {
			parts[1] = strings.Replace(parts[1], "refs/heads/", "", 1)
		} else if strings.HasPrefix(parts[1], "refs/remotes/") {
			parts[1] = strings.Replace(parts[1], "refs/remotes/", "", 1)
		} else {
			continue
		}
		branches = append(branches, parts[1])
	}

	return branches, nil
}

func (r *LocalRepository) NamedBranches(ref string) (map[string]string, error) {
	heads := make(map[string]string)

	output, err := r.ExecuteGitCommandQuiet("show-ref", "--no-tags", ref)
	if err != nil {
		return heads, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, " ")
		parts[1] = strings.Replace(parts[1], "refs/heads/", "", 1)
		parts[1] = strings.Replace(parts[1], "refs/remotes/", "", 1)
		heads[parts[1]] = parts[0]
	}

	return heads, nil
}

// CheckBranchesHaveRemotes returns branch names that exist only locally (no origin/)
func (r *LocalRepository) CheckBranchesHaveRemotes(branchNames []string) ([]string, error) {
	localOnlyBranches := []string{}

	for _, branchName := range branchNames {
		heads, err := r.NamedBranches(branchName)
		if err != nil {
			continue // Branch doesn't exist at all
		}

		remoteBranchName := fmt.Sprintf("origin/%s", branchName)
		_, hasRemote := heads[remoteBranchName]
		_, hasLocal := heads[branchName]

		if hasLocal && !hasRemote {
			localOnlyBranches = append(localOnlyBranches, branchName)
		}
	}

	return localOnlyBranches, nil
}

func (r *LocalRepository) MainBranchName() (string, error) {
	output, err := r.ExecuteGitCommandQuiet("rev-parse", "--abbrev-ref", "origin/HEAD")
	if err != nil {
		return "", fmt.Errorf("%s", output)
	}

	mainBranchName := strings.Replace(output, "origin/", "", 1)

	return mainBranchName, nil
}

func CheckedOutBranchName() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	checkedOutBranchName := strings.TrimSpace(string(stdout))

	return checkedOutBranchName, nil
}

func GetBranchStack(mainBranchName, checkedOutBranchName string) ([]string, error) {
	// Get list of known local branches
	heads, err := Heads()
	if err != nil {
		return []string{}, err
	}

	// Create lookup maps for known local branches
	shaToBranches := make(map[string]string)
	branches := []string{}
	for branch, sha := range heads {
		branches = append(branches, branch)
		shaToBranches[sha] = branch
	}

	mainSha, err := GetSha(mainBranchName)
	if err != nil {
		return []string{}, err
	}
	headSha, err := GetSha(checkedOutBranchName)
	if err != nil {
		return []string{}, err
	}

	if mainSha == headSha && mainBranchName != checkedOutBranchName {
		return []string{mainBranchName, checkedOutBranchName}, nil
	}

	// Find merge base for known local branches
	args := append([]string{"merge-base", "--octopus"}, branches...)
	cmd := exec.Command("git", args...)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return []string{}, err
	}
	mergeBase := strings.TrimSpace(string(stdout))

	// Only one branch exists
	if mergeBase == heads[mainBranchName] {
		return []string{mainBranchName}, nil
	}

	// Use merge base to find all parent->child relationships
	args = append([]string{"rev-list", "--parents"}, branches...)
	args = append(args, fmt.Sprintf("^%s~1", mergeBase))
	cmd = exec.Command("git", args...)
	stdout, err = cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return []string{}, err
	}

	// Create parent <-> child lookup maps
	parentToChild := make(map[string]string)
	childToParent := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, " ")
		parentToChild[parts[0]] = parts[1]
		childToParent[parts[1]] = parts[0]
	}

	// Build branch stack
	branchStack := []string{}

	// Find parents
	currentSha := headSha
	for {
		if branch, ok := shaToBranches[currentSha]; ok {
			branchStack = append([]string{branch}, branchStack...)
		}

		// Exit when we reach the main branch
		if currentSha == mainSha {
			break
		}

		if nextSha, ok := parentToChild[currentSha]; ok {
			currentSha = nextSha
		} else {
			break
		}
	}

	// Find children
	currentSha = headSha
	for {
		if branch, ok := shaToBranches[currentSha]; ok && currentSha != headSha {
			branchStack = append(branchStack, branch)
		}

		if nextSha, ok := childToParent[currentSha]; ok {
			currentSha = nextSha
		} else {
			break
		}
	}

	return branchStack, nil
}

func GetSha(branchName string) (string, error) {
	cmd := exec.Command("git", "rev-parse", branchName)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	sha := strings.TrimSpace(string(stdout))

	return sha, nil
}
