package config

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

// SyncTLM takes a mapping from the TLM and puts it down on disk so
// that a later converge run may act upon it.
func (c *Configurator) SyncTLM(tlm map[int]string) error {
	for team := range tlm {
		if _, ok := c.fc.Teams[team]; !ok {
			return fmt.Errorf("TLM requested an unknown team: %d", team)
		}
	}

	// This is a map of field number to map of port name to team VLAN ID
	fMap := make(map[int]map[string]int)

	for team, location := range tlm {
		parts := strings.Split(location, ":")
		fNum, _ := strconv.Atoi(strings.ReplaceAll(parts[0], "field", ""))
		if fMap[fNum] == nil {
			fMap[fNum] = make(map[string]int)
		}
		fMap[fNum][c.quadToEther(parts[1])] = c.fc.Teams[team].VLAN
	}

	f, err := os.Create(filepath.Join(c.stateDir, "tlm.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	f.Chmod(0644)

	if err := json.NewEncoder(f).Encode(fMap); err != nil {
		return err
	}

	return nil
}

// Init performs initialization of the underlying terraform workspace
// to fetch plugins and initialize module links.
func (c *Configurator) Init() error {
	cmd := exec.Command("terraform", "init")
	cmd.Dir = c.stateDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	return cmd.Wait()
}

// Converge commands all network hardware to achieve the state
// currently on disk.
func (c *Configurator) Converge(refresh bool, target string) error {
	opts := []string{"apply", "-auto-approve"}
	if !refresh {
		opts = append(opts, "-refresh=false")
	}
	if target != "" {
		opts = append(opts, "-target="+target)
	}
	cmd := exec.Command("terraform", opts...)
	cmd.Dir = c.stateDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	return cmd.Wait()
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

func (c *Configurator) quadToEther(quad string) string {
	switch quad {
	case "red":
		return "ether2"
	case "blue":
		return "ether3"
	case "green":
		return "ether4"
	case "yellow":
		return "ether5"
	}
	return ""
}
