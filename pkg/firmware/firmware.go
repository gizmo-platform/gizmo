package firmware

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// Config contains the information that a given firmware image will
// bake in.
type Config struct {
	ServerIP string
	NetSSID  string
	NetPSK   string
	Team     int
}

//go:embed src/* config.h.tpl
var f embed.FS

// RestoreToDir unpacks the firmware to a given directory
func RestoreToDir(dir string) error {
	// The source has to wind up in a folder called 'gizmo-fw'
	// because that's the name of the main file.  You can
	// technically override this, but this also means you can open
	// things in the Arduino IDE and it works.
	dir = filepath.Join(dir, "gizmo-fw")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	os.Create(filepath.Join(dir, "secrets.h"))
	return fs.WalkDir(f, "src", func(p string, d fs.DirEntry, _ error) error {
		if d.IsDir() {
			return nil
		}
		fo, err := os.Create(filepath.Join(dir, filepath.Base(p)))
		if err != nil {
			return err
		}
		data, _ := f.ReadFile(p)
		fo.Write(data)
		fo.Close()
		return nil
	})
}

// ConfigureBuild writes out a header file that contains remaining
// configuration elements for the build prior to invoking the
// compiler.
func ConfigureBuild(p string, cfg Config) error {
	tmpl, _ := template.ParseFS(f, "config.h.tpl")
	fo, err := os.Create(filepath.Join(p, "gizmo-fw", "params.h"))
	if err != nil {
		return err
	}
	defer fo.Close()
	return tmpl.Execute(fo, cfg)
}

// Build a configured source directory by calling arduino-cli.
// arduino-cli has an RPC mode, but its not super well tested and this
// is much easier to do on a quick timeline.  Ideally we'd instead
// just embed the arduino-cli into this binary which would also let us
// do things like programatically drive the board and library manager,
// but that's a task for a future iteration.
func Build(p string) error {
	args := []string{
		"compile",
		"--fqbn", "rp2040:rp2040:rpipicow",
		"--output-dir", ".",
		".",
	}
	cmd := exec.Command("arduino-cli", args...)
	cmd.Dir = filepath.Join(p, "gizmo-fw")

	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	return err
}

// CopyFirmware retrieves a copy of the compiled u2f file and puts it
// at the location specified by d.
func CopyFirmware(p, d string) error {
	p = filepath.Join(p, "gizmo-fw", "gizmo-fw.ino.uf2")
	src, err := os.Open(p)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(d)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
