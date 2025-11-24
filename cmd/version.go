package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"pila.olrik.dev/internal/core"
)

func NewVersionCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{},
		Short:   "Show version",
		Long:    `Show version`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, core.Version)
		},
	}

	return versionCmd
}
