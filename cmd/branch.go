package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewBranchListCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Show branch stack",
		Long:    "Show branch stack",
		Run: func(cmd *cobra.Command, args []string) {
			// mainBranchName, err := git.MainBranchName()
			// if err != nil {
			// 	return
			// }

			// checkedOutBranchName, err := git.CheckedOutBranchName()
			// if err != nil {
			// 	return
			// }

			// branchStack, err := git.GetBranchStack(mainBranchName, checkedOutBranchName)
			// if err != nil {
			// 	fmt.Println(err)
			// 	return
			// }

			// greenText := color.New(color.FgGreen).SprintFunc()
			// boldText := color.New(color.FgHiWhite, color.Bold).SprintFunc()
			// for i, branchName := range branchStack {
			// 	hereIndicator := "  "
			// 	if branchName == checkedOutBranchName {
			// 		hereIndicator = boldText(" *")
			// 		branchName = greenText(branchName)
			// 	}
			// 	fmt.Printf("%s%s%s\n", strings.Repeat("  ", i), branchName, hereIndicator)
			// }
		},
	}
	return listCmd
}

func NewBranchProposeCommand() *cobra.Command {
	proproseCmd := &cobra.Command{
		Use:   "propose",
		Short: "Publish & propose stack",
		Long:  "Publish & propose stack",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Publish & propose stack")
		},
	}
	return proproseCmd
}

func NewBranchPublishCommand() *cobra.Command {
	publishCmd := &cobra.Command{
		Use:   "publish",
		Short: "Publish stack",
		Long:  "Publish stack",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Publish stack")
		},
	}
	return publishCmd
}
