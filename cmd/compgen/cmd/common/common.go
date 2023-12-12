package cmd

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type TemplateData struct {
	Name        string
	PackageName string
}

// Executes `go mod tidy`
func Tidy(path string) {
	cmd := exec.Command("go", "mod", "tidy", "-e")
	cmd.Dir = path
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(string(output))
	}
}

func Render(source fs.FS, destination string, data TemplateData) error {
	// Iterate over all files/folders in 'source' and execute func() for entry found
	return fs.WalkDir(source, "templates", func(path string, entry fs.DirEntry, err error) error {
		// There was an error reading the fs. So we return the error to stop execution
		if err != nil {
			return err
		}

		// WalkDir starts with the root directory passed. So we skip the first iteration
		if path == "templates" {
			return nil
		}

		// We remove the top-level 'templates' directory
		// E.g. "templates/main.go.tmpl" becomes "main.go.tmpl"
		newFilename, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		newFilename = filepath.Join(destination, strings.TrimSuffix(newFilename, ".tmpl"))

		// Copy directories
		if entry.IsDir() {
			return os.MkdirAll(newFilename, os.ModePerm)
		}

		// Note that 'path' is not unmodified
		if strings.HasSuffix(path, ".tmpl") {
			content, err := fs.ReadFile(source, path)
			if err != nil {
				return err
			}

			file, err := os.Create(newFilename)
			if err != nil {
				return err
			}
			defer file.Close()

			// The template is finally rendered and immediately written to file
			tmpl := template.Must(template.New(newFilename).Parse(string(content)))
			return tmpl.Execute(file, data)
		}

		// A non-directory, non-template file. Ignore it
		return nil
	})
}
