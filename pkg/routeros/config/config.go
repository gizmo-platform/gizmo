package config

import (
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const (
	workspaceFile = "main.tf"
)

// New initializes and returns a configurator
func New(opts ...Option) *Configurator {
	c := new(Configurator)
	c.stateDir = ".netstate"
	c.routerAddr = "10.0.0.1"

	for _, o := range opts {
		o(c)
	}
	return c
}

// SyncState pushes the in-memory state down to the disk.
func (c *Configurator) SyncState(bootstrap bool) error {
	if err := os.MkdirAll(c.stateDir, 0755); err != nil {
		c.l.Warn("Couldn't make state directory", "error", err)
		return err
	}

	if err := c.extractModules(); err != nil {
		c.l.Warn("Couldn't extract modules", "error", err)
		return err
	}

	if err := c.configureWorkspace(bootstrap); err != nil {
		c.l.Warn("Couldn't configure workspace", "error", err)
		return err
	}

	if err := c.syncFMSConfig(); err != nil {
		c.l.Warn("Couldn't synchronize FMS config", "error", err)
		return err
	}

	return nil
}

// Converge commands all network hardware to achieve the state
// currently on disk.
func (c *Configurator) Converge(doInit bool) error {
	initCmd := exec.Command("terraform", "init")
	applyCmd := exec.Command("terraform", "apply", "-auto-approve", "-refresh=false")
	cmds := []*exec.Cmd{}
	if doInit {
		cmds = append(cmds, initCmd)
	}
	cmds = append(cmds, applyCmd)
	for _, cmd := range cmds {
		cmd.Dir = c.stateDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
		cmd.Wait()
	}
	return nil
}

func (c *Configurator) syncFMSConfig() error {
	f, err := os.Create(filepath.Join(c.stateDir, "fms.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(c.fc)
}

func (c *Configurator) extractModules() error {
	sub, _ := fs.Sub(efs, "tf/mod")
	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			c.l.Debug("Creating Directory", "dir", path)
			if err := os.MkdirAll(filepath.Join(c.stateDir, "mod", path), 0755); err != nil {
				return err
			}
		} else {
			c.l.Debug("Extracting File", "file", path)
			src, err := sub.Open(path)
			if err != nil {
				return err
			}
			defer src.Close()

			dst, err := os.Create(filepath.Join(c.stateDir, "mod", path))
			if err != nil {
				return err
			}
			defer dst.Close()
			if _, err := io.Copy(dst, src); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *Configurator) configureWorkspace(bootstrap bool) error {
	ctx := make(map[string]interface{})
	ctx["FMS"] = c.fc
	ctx["RouterAddr"] = c.routerAddr
	ctx["Bootstrap"] = bootstrap

	tmpl, err := template.New(workspaceFile).ParseFS(efs, filepath.Join("tf", workspaceFile))
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(c.stateDir, workspaceFile))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.Execute(f, ctx); err != nil {
		return err
	}

	return nil
}
