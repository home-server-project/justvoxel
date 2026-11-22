package server

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

// Use the same catalog-backed runtime as standalone Node tests, including VM
// fragments and eval. This does not replace any fixture's browser mocks.
func nodeTestCommand(args ...string) *exec.Cmd {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate WebUI test i18n prelude")
	}
	helper := filepath.Join(filepath.Dir(filename), "../../../tests/helpers/webui-i18n.cjs")
	return exec.Command("node", append([]string{"--require", helper}, args...)...)
}
