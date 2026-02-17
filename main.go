package main

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/text"
	"go.olrik.dev/pila/cmd"
)

func main() {
	text.EnableColors()

	root := cmd.NewRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
