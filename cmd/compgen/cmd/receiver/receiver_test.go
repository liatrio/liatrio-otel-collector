package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	// Missing args. Note: this is normally enforced by cobra
	args := []string{}
	assert.Panics(t, func() { run(ReceiverCmd, args) })

	// Missing outputDir argument
	args = []string{"github.com/dummy"}
	assert.Panics(t, func() { run(ReceiverCmd, args) })

	// Healthy run
	dir := t.TempDir()
	args = []string{"github.com/dummy", dir}
	assert.NotPanics(t, func() { run(ReceiverCmd, args) })

	// Validate file count
	// Note that the +1 is the result of generating go.sum without a .tmpl file
	entries, err := os.ReadDir(filepath.Join(dir, "dummy"))
	assert.NoError(t, err)
	sources, err := Templates.ReadDir("templates")
	assert.NoError(t, err)
	assert.Equal(t, len(sources)+1, len(entries))
}
