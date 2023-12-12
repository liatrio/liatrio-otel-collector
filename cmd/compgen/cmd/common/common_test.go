package cmd

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTidy(t *testing.T) {
	dir := t.TempDir()
	assert.Panics(t, func() { Tidy(dir) })

	cmd := exec.Command("go", "mod", "init", "dummy")
	cmd.Dir = dir
	cmd.Output()

	assert.NotPanics(t, func() { Tidy(dir) })
	assert.FileExists(t, filepath.Join(dir, "go.mod"))
}

