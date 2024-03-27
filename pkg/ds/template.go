package ds

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

func (ds *DriverStation) doTemplate(path, source string, mode fs.FileMode, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		ds.l.Error("Error creating target template path", "path", path, "error", err)
		return err
	}

	fMap := template.FuncMap{
		"ip4prefix": ip4prefix,
	}

	tmpl, err := template.New(filepath.Base(source)).Funcs(fMap).ParseFS(efs, source)
	if err != nil {
		ds.l.Error("Error parsing template", "source", source, "error", err)
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		ds.l.Error("Error creating target file", "file", path, "error", err)
		return err
	}
	defer f.Close()

	if err := f.Chmod(mode); err != nil {
		ds.l.Error("Could not set filemode", "file", path, "error", err)
	}

	if err := tmpl.Execute(f, data); err != nil {
		ds.l.Error("Error executing template", "data", data, "error", err)
		return err
	}

	return nil
}

func ip4prefix(t int) string {
	return fmt.Sprintf("10.%d.%d", int(t/100), t%100)
}
