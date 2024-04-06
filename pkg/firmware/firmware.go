package firmware

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-hclog"
)

// NewFactory vends a factory configured to use the included logger.
func NewFactory(l hclog.Logger) *Factory {
	return &Factory{l.Named("factory")}
}

func (f *Factory) unpack(bc BuildConfig) error {
	// The source has to wind up in a folder called 'firmware'
	// because that's the name of the main file.  You can
	// technically override this, but this also means you can open
	// things in the Arduino IDE and it works.
	dir := filepath.Join(bc.dir, "firmware")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	sf, err := os.Create(filepath.Join(dir, "secrets.h"))
	if err != nil {
		return err
	}
	sf.Close()
	return fs.WalkDir(efs, "src", func(p string, d fs.DirEntry, _ error) error {
		if d.IsDir() {
			return nil
		}
		fo, err := os.Create(filepath.Join(dir, filepath.Base(p)))
		if err != nil {
			return err
		}
		data, _ := efs.ReadFile(p)
		fo.Write(data)
		fo.Close()
		return nil
	})
}

func (f *Factory) configure(bc BuildConfig) error {
	tmpl, _ := template.ParseFS(efs, "config.h.tpl")
	fo, err := os.Create(filepath.Join(bc.dir, "firmware", "params.h"))
	if err != nil {
		return err
	}
	defer fo.Close()
	return tmpl.Execute(fo, bc.cfg)
}

// compile a configured source directory by calling arduino-cli.
// arduino-cli has an RPC mode, but its not super well tested and this
// is much easier to do on a quick timeline.  Ideally we'd instead
// just embed the arduino-cli into this binary which would also let us
// do things like programatically drive the board and library manager,
// but that's a task for a future iteration.
func (f *Factory) compile(bc BuildConfig) error {
	args := []string{
		"compile",
		"--profile", "gizmo",
		"--output-dir", ".",
		".",
	}
	cmd := exec.Command("arduino-cli", args...)
	cmd.Dir = filepath.Join(bc.dir, "firmware")

	output, err := cmd.CombinedOutput()
	f.l.Trace(string(output))
	return err
}

func (f *Factory) exportTo(bc BuildConfig) error {
	s := filepath.Join(bc.dir, "firmware", "firmware.ino.uf2")
	src, err := os.Open(s)
	if err != nil {
		f.l.Debug("Failed to open source", "path", s)
		return err
	}
	defer src.Close()

	dst, err := os.Create(bc.out)
	if err != nil {
		f.l.Debug("Failed to open target", "path", bc.out)
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (f *Factory) cleanup(bc BuildConfig) error {
	cleanFunc := func() error {
		return os.RemoveAll(bc.dir)
	}

	return backoff.Retry(cleanFunc, backoff.NewExponentialBackOff())
}

// Build runs the entire build pipeline with the provided options.
func (f *Factory) Build(opts ...BuildOption) error {
	b := BuildConfig{}

	for _, o := range opts {
		o(&b)
	}

	steps := []BuildStep{f.unpack, f.configure, f.compile, f.exportTo}
	names := []string{"Unpack", "Configure", "Compile", "Export"}
	if b.extractOnly {
		steps = []BuildStep{f.unpack, f.configure}
		names = []string{"Unpack", "Configure"}
		b.keep = true
	}
	if !b.keep {
		steps = append(steps, f.cleanup)
		names = append(names, "Cleanup")
	}
	for i, step := range steps {
		f.l.Info("Performing Step", "team", b.cfg.Team, "step", names[i])
		if err := step(b); err != nil {
			f.l.Warn("Halting build", "error", err)
			return err
		}
	}
	return nil
}
