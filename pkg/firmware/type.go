package firmware

import (
	"embed"

	"github.com/hashicorp/go-hclog"
)

// Config contains the information that a given firmware image will
// bake in.
type Config struct {
	Team       int
	UseConsole bool
	UseAvahi   bool
	UseExtNet  bool
	NetSSID    string
	NetPSK     string
	ServerIP   string
}

//go:embed src/*.cpp src/*.h src/*.ino src/*.yaml config.h.tpl
var efs embed.FS

// Factory binds all the build steps to a single struct in order to
// share the config.
type Factory struct {
	l hclog.Logger
}

// BuildConfig contains enough to build a single uf2 image.
type BuildConfig struct {
	dir         string
	cfg         Config
	keep        bool
	extractOnly bool
	out         string
}

// BuildOption configures the BuildConfig object.
type BuildOption func(*BuildConfig)

// BuildStep represents a single phase of the build.
type BuildStep func(BuildConfig) error
