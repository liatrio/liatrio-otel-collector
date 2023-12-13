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
	cmd.Output()

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
	os.Mkdir(filepath.Join(source, "templates"), os.ModePerm)
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)

	// Ignore  non-template files
	os.Create(filepath.Join(source, "templates", "main.go"))
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(destination, "main.go"))

	// Empty tmpl file
	file, _ := os.Create(filepath.Join(source, "templates", "main.go.tmpl"))
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(destination, "main.go"))

	// Normal tmpl file
	file.WriteString("pkg {{ .PackageName }}")
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(destination, "main.go"))
	content, _ := os.ReadFile(filepath.Join(destination, "main.go"))
	assert.Equal(t, "pkg github.com/dummy", string(content))

	// Empty tmpl file
	os.Mkdir(filepath.Join(source, "templates", "more"), os.ModePerm)
	os.Create(filepath.Join(source, "templates", "more", "test.go.tmpl"))
	err = Render(os.DirFS(source), destination, data)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(destination, "more", "test.go"))
}
