package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	// Missing args
	// Note that this should be impossible at runtime. We use `cobra.MinimumNArgs()`
	// when building `ReceiverCmd` to enforce that the correct number of arguments
	// are passed in at runtime. This is enforced outside of run() so we still want
	// to test the scenario.
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
	// Note that the +1 is the result of generating go.sum without a go.sum.tmpl file
	entries, err := os.ReadDir(filepath.Join(dir, "dummy"))
	assert.NoError(t, err)
	sources, err := templates.ReadDir("templates")
	assert.NoError(t, err)
	assert.Equal(t, len(sources)+1, len(entries))
}
