package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	PILA_HOOK_DIRECTORY            = ".pila.hooks.d"
	PILA_HOOK_MULTI_MERGE_COMPLETE = "multi-merge-completed.sh"
)

func (r *LocalRepository) RunHook(hookFile string, arg ...string) error {

	hookFile, err := filepath.Abs(filepath.Join(PILA_HOOK_DIRECTORY, hookFile))
	if err != nil {
		return err
	}

	_, err = os.Stat(hookFile)
	if err == nil {
		r.Note("Running hook after all merges completed successfully")
		cmd := exec.Command(hookFile, arg...)
		output, err := cmd.Output()
		if err != nil {
			return err
		}
		fmt.Println(strings.TrimSpace(string(output)))
	}

	return nil
}
