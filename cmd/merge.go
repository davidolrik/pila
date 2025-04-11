package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"pila.dev/pila/internal/git"
)

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

			appendReferences, _ := cmd.Flags().GetBool("append")
			prependReferences, _ := cmd.Flags().GetBool("prepend")
			if len(branches) > 0 {
				if appendReferences || prependReferences {
					manifest, err := git.LoadMultiMergeManifest()
					cobra.CheckErr(err)
					if manifest.Type != git.MULTI_MERGE_MANIFEST_TYPE_BRANCHES {
						err := fmt.Errorf("manifest is not of type branches")
						cobra.CheckErr(err)
					}

					existingBranches := []string{}
					for _, reference := range manifest.References {
						existingBranches = append(existingBranches, reference.Name)
					}

					if appendReferences {
						branches = append(existingBranches, branches...)
					} else if prependReferences {
						branches = append(branches, existingBranches...)
					}
				}

				_, err := repo.MultiMergeNamedBranches(target, branches)
				cobra.CheckErr(err)
			} else if len(labels) > 0 {
				err := repo.MultiMergeNamedLabels(target, labels)
				cobra.CheckErr(err)
			} else {
				err := repo.MultiMergeUsingManifest()
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

	multiMergeCmd.Flags().BoolP("append", "A", false, "Add reference to end of existing manifest")
	multiMergeCmd.Flags().BoolP("prepend", "P", false, "Add reference to start of existing manifest")
	multiMergeCmd.Flags().Bool("redo", false, "Redo stack using existing manifest")

	multiMergeCmd.MarkFlagsMutuallyExclusive("append", "prepend", "redo")

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

	multiMergeCmd.MarkFlagsOneRequired("branch", "label", "append", "prepend", "redo")

	multiMergeCmd.AddCommand(NewMultiMergeAbortCommand())
	multiMergeCmd.AddCommand(NewMultiMergeContinueCommand())
	multiMergeCmd.AddCommand(NewMultiMergeShowCommand())

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
