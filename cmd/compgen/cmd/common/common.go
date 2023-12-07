package cmd

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const PackageDir = "../../pkg"

type TemplateData struct {
	Name        string
	PackageName string
}

func CompleteModule(modulePath string) {
	cmd := exec.Command("go", "mod", "tidy", "-e")
	cmd.Dir = modulePath
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(string(output))
	}

	cmd = exec.Command("make", "gen")
	cmd.Dir = modulePath
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatal(string(output))
	}
}

func RenderTemplates(templateDir string, destinationDir string, templateData TemplateData) {
	if len(templateDir) == 0 {
		log.Fatal("the template directory cannot be an empty string")
	}

	if len(destinationDir) == 0 {
		log.Fatal("the destination directory cannot be an empty string")
	}

	templateDir = filepath.Clean(templateDir)
	destinationDir = filepath.Clean(destinationDir)

	// Iterate over all files/folders in templateDir and execute func() for entry found
	err := filepath.WalkDir(templateDir, func(path string, entry os.DirEntry, err error) error {
		// There was an error reading the directory. So we return the error to stop execution
		if err != nil {
			return err
		}

		// filepath.WalkDir starts with the root directory passed. So we skip the first iteration
		if path == templateDir {
			return nil
		}

		// Copy directories
		if entry.IsDir() {
			return os.Mkdir(strings.Replace(path, templateDir, destinationDir, 1), os.ModePerm)
		}

		// Ignore non-template files
		if !strings.HasSuffix(entry.Name(), ".tmpl") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl := template.Must(template.New(path).Parse(string(content)))

		path = strings.TrimSuffix(path, ".tmpl")
		path = strings.Replace(path, templateDir, destinationDir, 1)
		file, err := os.Create(path)
		if err != nil {
			return err
		}

		defer file.Close()

		// The template is finally rendered and immediately written to file
		return tmpl.Execute(file, templateData)
	})

	if err != nil {
		log.Fatal(err)
	}
}
