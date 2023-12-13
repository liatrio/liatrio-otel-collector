package cmd

import (
	"os"
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
	_, err := cmd.Output()
	assert.NoError(t, err)

	assert.NotPanics(t, func() { Tidy(dir) })
	assert.FileExists(t, filepath.Join(dir, "go.mod"))
}

func TestRender(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	data := TemplateData{Name: "dummy", PackageName: "github.com/dummy"}

	// Source does not contain a "templates" directory
	err := Render(os.DirFS(source), destination, data)
	assert.Error(t, err)

	// Nothing to do
	err = os.Mkdir(filepath.Join(source, "templates"), os.ModePerm)
	assert.NoError(t, err)
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)

	// Ignore  non-template files
	_, err = os.Create(filepath.Join(source, "templates", "main.go"))
	assert.NoError(t, err)
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(destination, "main.go"))

	// Empty tmpl file
	file, err := os.Create(filepath.Join(source, "templates", "main.go.tmpl"))
	assert.NoError(t, err)
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(destination, "main.go"))

	// Normal tmpl file
	_, err = file.WriteString("pkg {{ .PackageName }}")
	assert.NoError(t, err)
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(destination, "main.go"))
	content, err := os.ReadFile(filepath.Join(destination, "main.go"))
	assert.NoError(t, err)
	assert.Equal(t, "pkg github.com/dummy", string(content))

	// Empty tmpl file
	err = os.Mkdir(filepath.Join(source, "templates", "more"), os.ModePerm)
	assert.NoError(t, err)
	_, err = os.Create(filepath.Join(source, "templates", "more", "test.go.tmpl"))
	assert.NoError(t, err)
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(destination, "more", "test.go"))
}
