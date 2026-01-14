package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"pila.olrik.dev/internal/git"
)

func handleMultiMergeError(err error) {
	if err == nil {
		return
	}

	// Check if this is a merge conflict error
	var conflictErr *git.MultiMergeConflictError
	if errors.As(err, &conflictErr) {
		fmt.Println()
		fmt.Println(color.RedString("Merge conflict detected!"))
		fmt.Println()
		fmt.Printf("A merge conflict occurred while merging branch %s\n", color.CyanString(conflictErr.BranchName))
		fmt.Println()
		fmt.Println("To resolve:")
		fmt.Println("  1. Fix the conflicts in your working directory")
		fmt.Println("  2. Stage the resolved files with " + color.GreenString("git add <files>"))
		fmt.Println("  3. Continue the multi-merge with " + color.GreenString("pila multi-merge continue"))
		fmt.Println()
		fmt.Println("Or abort the multi-merge with " + color.YellowString("pila multi-merge abort"))
		fmt.Println()
	}

	// Check if this is a local-only branches error
	var localOnlyErr *git.LocalOnlyBranchesError
	if errors.As(err, &localOnlyErr) {
		fmt.Println()
		fmt.Println(color.RedString("Cannot redo: some branches only exist locally"))
		fmt.Println()
		fmt.Println("The following branches have no remote tracking branch:")
		for _, branchName := range localOnlyErr.BranchNames {
			fmt.Printf("  - %s\n", color.CyanString(branchName))
		}
		fmt.Println()
		fmt.Println("To resolve, either:")
		fmt.Println("  1. Push the branch(es) to remote:")
		for _, branchName := range localOnlyErr.BranchNames {
			fmt.Printf("     %s\n", color.GreenString("git push -u origin %s", branchName))
		}
		fmt.Println()
		fmt.Println("  2. Remove the branch(es) from the manifest:")
		for _, branchName := range localOnlyErr.BranchNames {
			fmt.Printf("     %s\n", color.YellowString("pila multi-merge remove %s", branchName))
		}
		fmt.Println()
	}
}

func checkOngoingMerge(repo *git.LocalRepository) error {
	if branchName, err := repo.OngoingMergeBranchName(); err == nil && branchName != "" {
		return fmt.Errorf("a merge is currently in progress, please run 'pila multi-merge continue' or 'pila multi-merge abort' first")
	}
	return nil
}

func branchNameCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	repo, err := git.GetLocalRepository()
	if err != nil {
		panic(err)
	}

	// All branch names, both local and remote
	suggestions := []string{}
	validOptions, err := repo.AllBranchNames()
	if err != nil {
		return suggestions, cobra.ShellCompDirectiveNoFileComp
	}

	// Filter suggestions based on what the user has typed
	for _, option := range validOptions {
		if strings.HasPrefix(option, toComplete) {
			suggestions = append(suggestions, option)
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

func NewMultiMergeCommand() *cobra.Command {
	multiMergeCmd := &cobra.Command{
		Use:     "multi-merge",
		Aliases: []string{"mm"},
		Short:   "Merge multiple branches into single target branch",
		Long: strings.TrimSpace(dedent.Dedent(`
			Merge multiple branches into single target branch

			Select branches to merge based either on branch names or Merge Request labels
		`)),
		Run: func(cmd *cobra.Command, args []string) {
			branches, _ := cmd.Flags().GetStringSlice("branch")
			labels, _ := cmd.Flags().GetStringSlice("label")
			target, _ := cmd.Flags().GetString("target")

			// Get handle on local repo
			repo, err := git.GetLocalRepository()
			if err != nil {
				panic(err)
			}

			// Check if there's an ongoing merge
			err = checkOngoingMerge(repo)
			cobra.CheckErr(err)

			// Load previous manifest if needed
			manifest, _ := git.LoadMultiMergeManifest()
			if target == "" && manifest != nil {
				target = manifest.Target
			}

			// All multi merges require a target branch
			if (len(branches) > 0 || len(labels) > 0) && target == "" {
				err := errors.New("target is required when specifying branches or labels")
				cobra.CheckErr(err)
			}

			if len(branches) > 0 {
				_, err := repo.MultiMergeNamedBranches(target, branches)
				handleMultiMergeError(err)
				cobra.CheckErr(err)
			} else if len(labels) > 0 {
				err := repo.MultiMergeNamedLabels(target, labels)
				cobra.CheckErr(err)
			}
		},
	}
	multiMergeCmd.Flags().StringP("target", "T", "", strings.TrimSpace(dedent.Dedent(`
			Target branch
			⚠️ the target branch will be deleted and recreated locally as a clean branch off of the main branch
		`)),
	)
	multiMergeCmd.RegisterFlagCompletionFunc("target", branchNameCompletions)
	multiMergeCmd.Flags().BoolP("force", "F", false, "Don't ask before deleting target branch")

	multiMergeCmd.Flags().StringSliceP("branch", "B", []string{}, strings.TrimSpace(dedent.Dedent(`
			Branches to merge into target branch
			Note that order matters
		`)),
	)
	multiMergeCmd.RegisterFlagCompletionFunc("branch", branchNameCompletions)

	multiMergeCmd.Flags().StringSliceP("label", "L", []string{}, strings.TrimSpace(dedent.Dedent(`
			Labels on Merge requests to merge into target branch
			Merge requests will be merged in order of their creation date
		`)),
	)

	multiMergeCmd.MarkFlagsMutuallyExclusive("branch", "label")

	multiMergeCmd.MarkFlagsOneRequired("branch", "label")

	multiMergeCmd.AddCommand(NewMultiMergeAbortCommand())
	multiMergeCmd.AddCommand(NewMultiMergeContinueCommand())
	multiMergeCmd.AddCommand(NewMultiMergeShowCommand())
	multiMergeCmd.AddCommand(NewMultiMergeRedoCommand())
	multiMergeCmd.AddCommand(NewMultiMergeAppendCommand())
	multiMergeCmd.AddCommand(NewMultiMergePrependCommand())
	multiMergeCmd.AddCommand(NewMultiMergeRemoveCommand())

	return multiMergeCmd
}

func NewMultiMergeContinueCommand() *cobra.Command {
	multiMergeContinueCmd := &cobra.Command{
		Use:     "continue",
		Aliases: []string{"cont"},
		Short:   "Continue ongoing Multi Merge operation",
		Long: strings.TrimSpace(dedent.Dedent(`
		`)),
		Run: func(cmd *cobra.Command, args []string) {
			// Get handle on local repo
			repo, err := git.GetLocalRepository()
			if err != nil {
				panic(err)
			}

			for {
				manifest, err := repo.MultiMergeNamedContinue()
				handleMultiMergeError(err)
				cobra.CheckErr(err)
				if manifest.IsDone() {
					break
				}
			}
		},
	}
	return multiMergeContinueCmd
}

func NewMultiMergeAbortCommand() *cobra.Command {
	multiMergeAbortCmd := &cobra.Command{
		Use:     "abort",
		Aliases: []string{},
		Short:   "Abort ongoing Multi Merge operation",
		Long: strings.TrimSpace(dedent.Dedent(`
		`)),
		Run: func(cmd *cobra.Command, args []string) {
			// Get handle on local repo
			repo, err := git.GetLocalRepository()
			if err != nil {
				panic(err)
			}

			err = repo.MultiMergeAbort()
			cobra.CheckErr(err)
		},
	}
	return multiMergeAbortCmd
}

func NewMultiMergeShowCommand() *cobra.Command {
	multiMergeShowCmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{},
		Short:   "Show current Multi merge status",
		Long: strings.TrimSpace(dedent.Dedent(`
		`)),
		Run: func(cmd *cobra.Command, args []string) {
			// Get handle on local repo
			manifest, err := git.LoadMultiMergeManifest()
			cobra.CheckErr(err)

			for _, reference := range manifest.References {
				status := color.RedString("Not merged")
				if reference.Merged {
					status = color.GreenString("Merged")
				}

				fmt.Printf("%s %s\n", color.CyanString("%s", reference.Name), status)
			}
		},
	}
	return multiMergeShowCmd
}

func NewMultiMergeRedoCommand() *cobra.Command {
	multiMergeRedoCmd := &cobra.Command{
		Use:     "redo",
		Aliases: []string{},
		Short:   "Redo multi-merge using existing manifest",
		Long: strings.TrimSpace(dedent.Dedent(`
			Load the existing manifest and reapply all merges from scratch.
			This resets the target branch to the main branch and re-merges all branches in order.
		`)),
		Run: func(cmd *cobra.Command, args []string) {
			// Get handle on local repo
			repo, err := git.GetLocalRepository()
			if err != nil {
				panic(err)
			}

			// Check if there's an ongoing merge
			err = checkOngoingMerge(repo)
			cobra.CheckErr(err)

			err = repo.MultiMergeUsingManifest()
			handleMultiMergeError(err)
			cobra.CheckErr(err)
		},
	}
	return multiMergeRedoCmd
}

func NewMultiMergeAppendCommand() *cobra.Command {
	multiMergeAppendCmd := &cobra.Command{
		Use:     "append",
		Aliases: []string{},
		Short:   "Append branches to existing multi-merge manifest",
		Long: strings.TrimSpace(dedent.Dedent(`
			Add branches to the end of the existing multi-merge manifest and merge them.
		`)),
		Run: func(cmd *cobra.Command, args []string) {
			branches, _ := cmd.Flags().GetStringSlice("branch")
			target, _ := cmd.Flags().GetString("target")

			// Get handle on local repo
			repo, err := git.GetLocalRepository()
			if err != nil {
				panic(err)
			}

			// Check if there's an ongoing merge
			err = checkOngoingMerge(repo)
			cobra.CheckErr(err)

			// Load existing manifest
			manifest, err := git.LoadMultiMergeManifest()
			cobra.CheckErr(err)

			if manifest.Type != git.MULTI_MERGE_MANIFEST_TYPE_BRANCHES {
				err := fmt.Errorf("manifest is not of type branches")
				cobra.CheckErr(err)
			}

			// Use target from manifest if not specified
			if target == "" {
				target = manifest.Target
			}

			// Get existing branches from manifest
			existingBranches := []string{}
			for _, reference := range manifest.References {
				existingBranches = append(existingBranches, reference.Name)
			}

			// Append new branches to existing ones
			allBranches := append(existingBranches, branches...)

			_, err = repo.MultiMergeNamedBranches(target, allBranches)
			handleMultiMergeError(err)
			cobra.CheckErr(err)
		},
	}
	multiMergeAppendCmd.Flags().StringSliceP("branch", "B", []string{}, strings.TrimSpace(dedent.Dedent(`
		Branches to append to the existing manifest
	`)))
	multiMergeAppendCmd.RegisterFlagCompletionFunc("branch", branchNameCompletions)
	multiMergeAppendCmd.MarkFlagRequired("branch")

	multiMergeAppendCmd.Flags().StringP("target", "T", "", "Target branch (inherits from manifest if not specified)")
	multiMergeAppendCmd.RegisterFlagCompletionFunc("target", branchNameCompletions)

	return multiMergeAppendCmd
}

func NewMultiMergePrependCommand() *cobra.Command {
	multiMergePrependCmd := &cobra.Command{
		Use:     "prepend",
		Aliases: []string{},
		Short:   "Prepend branches to existing multi-merge manifest",
		Long: strings.TrimSpace(dedent.Dedent(`
			Add branches to the start of the existing multi-merge manifest and merge them.
		`)),
		Run: func(cmd *cobra.Command, args []string) {
			branches, _ := cmd.Flags().GetStringSlice("branch")
			target, _ := cmd.Flags().GetString("target")

			// Get handle on local repo
			repo, err := git.GetLocalRepository()
			if err != nil {
				panic(err)
			}

			// Check if there's an ongoing merge
			err = checkOngoingMerge(repo)
			cobra.CheckErr(err)

			// Load existing manifest
			manifest, err := git.LoadMultiMergeManifest()
			cobra.CheckErr(err)

			if manifest.Type != git.MULTI_MERGE_MANIFEST_TYPE_BRANCHES {
				err := fmt.Errorf("manifest is not of type branches")
				cobra.CheckErr(err)
			}

			// Use target from manifest if not specified
			if target == "" {
				target = manifest.Target
			}

			// Get existing branches from manifest
			existingBranches := []string{}
			for _, reference := range manifest.References {
				existingBranches = append(existingBranches, reference.Name)
			}

			// Prepend new branches before existing ones
			allBranches := append(branches, existingBranches...)

			_, err = repo.MultiMergeNamedBranches(target, allBranches)
			handleMultiMergeError(err)
			cobra.CheckErr(err)
		},
	}
	multiMergePrependCmd.Flags().StringSliceP("branch", "B", []string{}, strings.TrimSpace(dedent.Dedent(`
		Branches to prepend to the existing manifest
	`)))
	multiMergePrependCmd.RegisterFlagCompletionFunc("branch", branchNameCompletions)
	multiMergePrependCmd.MarkFlagRequired("branch")

	multiMergePrependCmd.Flags().StringP("target", "T", "", "Target branch (inherits from manifest if not specified)")
	multiMergePrependCmd.RegisterFlagCompletionFunc("target", branchNameCompletions)

	return multiMergePrependCmd
}

func manifestBranchCompletions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Load manifest to get branches
	manifest, err := git.LoadMultiMergeManifest()
	if err != nil || manifest == nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	suggestions := []string{}
	for _, reference := range manifest.References {
		if strings.HasPrefix(reference.Name, toComplete) {
			suggestions = append(suggestions, reference.Name)
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

func NewMultiMergeRemoveCommand() *cobra.Command {
	multiMergeRemoveCmd := &cobra.Command{
		Use:   "remove <branch>",
		Short: "Remove a branch from the multi-merge manifest",
		Long: strings.TrimSpace(dedent.Dedent(`
			Remove a branch from the existing multi-merge manifest.
			The manifest is modified but not committed.
		`)),
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: manifestBranchCompletions,
		Run: func(cmd *cobra.Command, args []string) {
			branchToRemove := args[0]

			// Load existing manifest
			manifest, err := git.LoadMultiMergeManifest()
			cobra.CheckErr(err)

			// Find and remove the branch
			found := false
			for i, reference := range manifest.References {
				if reference.Name == branchToRemove {
					manifest.References = append(manifest.References[:i], manifest.References[i+1:]...)
					found = true
					break
				}
			}

			if !found {
				err := fmt.Errorf("branch %s not found in manifest", branchToRemove)
				cobra.CheckErr(err)
			}

			// Save the manifest (without committing)
			err = manifest.Save()
			cobra.CheckErr(err)

			fmt.Printf("Removed %s from manifest\n", color.CyanString(branchToRemove))
		},
	}
	return multiMergeRemoveCmd
}
