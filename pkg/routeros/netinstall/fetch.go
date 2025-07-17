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

// Fetcher binds the methods that fetch packages, and allows the FMS
// to consume all parts of the fetching process as an interface.
type Fetcher struct {
	l hclog.Logger
}

// NewFetcher initializes the fetcher.
func NewFetcher(l hclog.Logger) *Fetcher {
	f := &Fetcher{l: l}
	if l == nil {
		f.l = hclog.NewNullLogger()
	}
	return f
}

// FetchPackages retrieves routeros packages that have been qualified
// and unpacks them into ImagePath.
func (f *Fetcher) FetchPackages() error {
	pkgs := []string{
		RouterPkgARM,
		RouterPkgARM64,
		RouterPkgMIPSBE,
		WifiPkgARM,
		WifiPkgARM64,
		WifiPkgMIPSBE,
	}

	if err := os.MkdirAll(ImagePath, 0755); err != nil {
		f.l.Error("Could not create image path", "error", err)
		return err
	}

	for _, pkg := range pkgs {
		f.l.Info("Fetching Package", "pkg", pkg)
		dest, err := os.Create(filepath.Join(ImagePath, pkg))
		if err != nil {
			f.l.Error("Error creating path", "error", err)
			return err
		}
		defer dest.Close()

		resp, err := http.Get(routerosCDN + pkg)
		if err != nil {
			f.l.Error("Error retrieving package", "error", err)
			return err
		}
		defer resp.Body.Close()

		if _, err := io.Copy(dest, resp.Body); err != nil {
			f.l.Error("Error writing package to disk", "error", err)
			return err
		}
	}

	return nil
}

// FetchTools retrieves the qualified version of netinstall-cli and
// unpacks it into netinstallPath
func (f *Fetcher) FetchTools() error {
	resp, err := http.Get(routerosCDN + netinstallPkg)
	if err != nil {
		f.l.Error("Error downloading", "error", err)
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
			dest, err := os.Create(netinstallPath)
			if err != nil {
				f.l.Error("Error creating file", "error", err)
				return err
			}
			defer dest.Close()
			dest.Chmod(0755)

			if _, err := io.Copy(dest, r); err != nil {
				f.l.Error("Error writing files", "error", err)
				return err
			}

			if err := exec.Command("/usr/bin/setcap", "cap_net_raw,cap_net_bind_service+ep", netinstallPath).Run(); err != nil {
				f.l.Error("Error elevating capability on tool", "error", err)
				return err
			}
			return nil
		}
	}

	return nil
}
