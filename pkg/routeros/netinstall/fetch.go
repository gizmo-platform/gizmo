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

	"github.com/gizmo-platform/gizmo/pkg/eventstream"
)

const (
	// This is the root path to download from.
	routerosCDN = "https://cdn.mikrotik.com/routeros/" + RouterOSVersion + "/"
)

// EventStreamer publishes events related to the file fetcher to the
// event bus.
type EventStreamer interface {
	PublishError(error)
	PublishFileFetch(string)
	PublishActionStart(string, string)
}

// Fetcher binds the methods that fetch packages, and allows the FMS
// to consume all parts of the fetching process as an interface.
type Fetcher struct {
	l hclog.Logger

	es EventStreamer

	packageDir string
	binDir     string
}

// FetcherOpt binds variadic setters that mutate the fetcher during
// initialization.
type FetcherOpt func(*Fetcher)

// WithFetcherLogger sets the logger for the fetcher.
func WithFetcherLogger(l hclog.Logger) FetcherOpt {
	return func(f *Fetcher) {
		f.l = l
	}
}

// WithFetcherEventStreamer sets the event streaming interface for the
// fetcher.
func WithFetcherEventStreamer(es EventStreamer) FetcherOpt {
	return func(f *Fetcher) {
		f.es = es
	}
}

// WithFetcherPackageDir allows changing the package directory
func WithFetcherPackageDir(dir string) FetcherOpt {
	return func(f *Fetcher) {
		f.packageDir = dir
	}
}

// WithFetcherBinDir allows changing the bin directory
func WithFetcherBinDir(dir string) FetcherOpt {
	return func(f *Fetcher) {
		f.binDir = dir
	}
}

// NewFetcher initializes the fetcher.
func NewFetcher(opts ...FetcherOpt) *Fetcher {
	f := new(Fetcher)
	for _, o := range opts {
		o(f)
	}
	if f.l == nil {
		f.l = hclog.NewNullLogger()
	}
	if f.es == nil {
		f.es = eventstream.NewNullStreamer()
	}
	if f.packageDir == "" {
		f.packageDir = ImagePath
	}
	if f.binDir == "" {
		f.binDir = BinPath
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

	if err := os.MkdirAll(f.packageDir, 0755); err != nil {
		f.l.Error("Could not create image path", "error", err)
		f.es.PublishError(err)
		return err
	}

	for _, pkg := range pkgs {
		f.l.Info("Fetching Package", "pkg", pkg)
		f.es.PublishActionStart("File Download", pkg)
		dest, err := os.Create(filepath.Join(f.packageDir, pkg))
		if err != nil {
			f.l.Error("Error creating path", "error", err)
			f.es.PublishError(err)
			return err
		}
		defer dest.Close()

		resp, err := http.Get(routerosCDN + pkg)
		if err != nil {
			f.l.Error("Error retrieving package", "error", err)
			f.es.PublishError(err)
			return err
		}
		defer resp.Body.Close()

		if _, err := io.Copy(dest, resp.Body); err != nil {
			f.l.Error("Error writing package to disk", "error", err)
			f.es.PublishError(err)
			return err
		}
		f.es.PublishFileFetch(pkg)
	}

	return nil
}

// FetchTools retrieves the qualified version of netinstall-cli and
// unpacks it into netinstallPath
func (f *Fetcher) FetchTools() error {
	f.l.Info("Downloading Mikrotik Tools")
	f.es.PublishActionStart("File Download", netinstallPkg)

	if err := os.MkdirAll(f.binDir, 0755); err != nil {
		f.l.Error("Could not create bin path", "error", err)
		f.es.PublishError(err)
		return err
	}

	resp, err := http.Get(routerosCDN + netinstallPkg)
	if err != nil {
		f.l.Error("Error downloading", "error", err)
		f.es.PublishError(err)
		return err
	}
	defer resp.Body.Close()

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		f.es.PublishError(err)
		return err
	}

	r := tar.NewReader(gr)
loop:
	for {
		hdr, err := r.Next()
		switch {
		case err == io.EOF:
			f.l.Debug("End of file during tar extraction")
			break loop
		case err != nil:
			f.es.PublishError(err)
			return err
		case hdr == nil:
			continue // handles rare bugs from broken source tar
		case hdr.Name == "netinstall-cli":
			dest, err := os.Create(netinstallPath)
			if err != nil {
				f.l.Error("Error creating file", "error", err)
				f.es.PublishError(err)
				return err
			}
			defer dest.Close()
			dest.Chmod(0755)

			if _, err := io.Copy(dest, r); err != nil {
				f.l.Error("Error writing files", "error", err)
				f.es.PublishError(err)
				return err
			}

			if err := exec.Command("sudo", "/usr/bin/setcap", "cap_net_raw,cap_net_bind_service+ep", netinstallPath).Run(); err != nil {
				f.l.Error("Error elevating capability on tool", "error", err)
				f.es.PublishError(err)
				return err
			}
			f.l.Info("Fetched Mikrotik Tool", "tool", "netinstall-cli")
			f.es.PublishFileFetch(netinstallPkg)
			return nil
		}
	}
	f.l.Warn("Escaped tool fetch without loading netinstall-cli!?")
	return nil
}
