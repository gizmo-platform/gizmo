package netinstall

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
)

const (
	// This is the root path to download from.
	routerosCDN = "https://cdn.mikrotik.com/routeros/" + RouterOSVersion + "/"
)

// FetchPackages retrieves routeros packages that have been qualified
// and unpacks them into ImagePath.
func FetchPackages(l hclog.Logger) error {
	pkgs := []string{
		RouterPkgARM,
		RouterPkgMIPSBE,
		WifiPkgARM,
		WifiPkgMIPSBE,
	}

	if err := os.MkdirAll(ImagePath, 0755); err != nil {
		l.Error("Could not create image path", "error", err)
		return err
	}

	for _, pkg := range pkgs {
		l.Info("Fetching Package", "pkg", pkg)
		f, err := os.Create(filepath.Join(ImagePath, pkg))
		if err != nil {
			l.Error("Error creating path", "error", err)
			return err
		}
		defer f.Close()

		resp, err := http.Get(routerosCDN + pkg)
		if err != nil {
			l.Error("Error retrieving package", "error", err)
			return err
		}
		defer resp.Body.Close()

		if _, err := io.Copy(f, resp.Body); err != nil {
			l.Error("Error writing package to disk", "error", err)
			return err
		}
	}

	return nil
}

// FetchTools retrieves the qualified version of netinstall-cli and
// unpacks it into netinstallPath
func FetchTools(l hclog.Logger) error {
	resp, err := http.Get(routerosCDN + netinstallPkg)
	if err != nil {
		l.Error("Error downloading", "error", err)
		return err
	}
	defer resp.Body.Close()

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	r := tar.NewReader(gr)
loop:
	for {
		hdr, err := r.Next()
		switch {
		case err == io.EOF:
			break loop
		case err != nil:
			return err
		case hdr == nil:
			continue // handles rare bugs from broken source tar
		case hdr.Name == "netinstall-cli":
			f, err := os.Create(netinstallPath)
			if err != nil {
				l.Error("Error creating file", "error", err)
				return err
			}
			defer f.Close()
			f.Chmod(0755)

			if _, err := io.Copy(f, r); err != nil {
				l.Error("Error writing files", "error", err)
				return err
			}

			if err := exec.Command("/usr/bin/setcap", "cap_net_bind_service+ep", netinstallPath).Run(); err != nil {
				l.Error("Error elevating capability on tool", "error", err)
				return err
			}
			return nil
		}
	}

	return nil
}
