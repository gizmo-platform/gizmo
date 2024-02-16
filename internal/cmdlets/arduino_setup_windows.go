package cmdlets

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// We need to mess with the path on windows to see if the user hasn't
// got the arduino-cli installed and we need to be able to reach into
// the one that's inside the Arduino IDE.  Yes, this is really the
// wrong way to do this, but it works and was fast to implement.
func init() {
	if _, err := exec.LookPath("arduino-cli"); err == nil {
		// If the arduino-cli exists in the normal path, don't
		// bother with any of this.
		return
	}

	dir, _ := os.UserCacheDir()
	arduinoPath := filepath.Join(
		dir,
		"programs", "Arduino IDE", "resources", "app", "lib", "backend", "resources",
	)

	_, err := os.Stat(arduinoPath)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}

	paths := append(filepath.SplitList(os.Getenv("PATH")), arduinoPath)
	os.Setenv("PATH", strings.Join(paths, string(os.PathListSeparator)))
}
