package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"pila.dev/pila/internal/core"
)

func NewRootCommand() *cobra.Command {
	var configPath string
	var verbose int

	homeDir, _ := os.UserHomeDir()

	rootCmd := &cobra.Command{
		Use:   "pila",
		Short: "Pila - Stack all the things!",
		Long:  `Pila - Stack all the things!`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize config and bind global flags to the config
			messages, err := core.InitializeConfig(cmd)
			for _, message := range messages {
				fmt.Println(message)
			}
			return err
		},
	}
	rootCmd.PersistentFlags().StringVar(
		&configPath, "config-path", fmt.Sprintf("%s/.config/pila", homeDir),
		"config path",
	)
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "more output, repeat for even more")

	debugCmd := &cobra.Command{
		Use:    "debug",
		Short:  "Debug command",
		Long:   "Debug command",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Debug")
		},
	}
	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(
		NewMultiMergeCommand(),
	)

	return rootCmd
}
