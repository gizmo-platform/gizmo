package firmware

import (
	"fmt"
	"os"
	"path/filepath"
)

// WithBuildDir sets the directory for building.  Generally this
// should be an ephemeral directory and you should be using
// WithEphemeralBuildDir instead.
func WithBuildDir(path string) BuildOption {
	return func(bc *BuildConfig) {
		bc.dir = path
	}
}

// WithEphemeralBuildDir creates and configures the build directory.
func WithEphemeralBuildDir() BuildOption {
	return func(bc *BuildConfig) {
		bc.dir, _ = os.MkdirTemp("", "gizmo")
	}
}

// WithBuildOutputFile either sets the output file or automatically
// generates the path from the build config.
func WithBuildOutputFile(path string) BuildOption {
	return func(bc *BuildConfig) {
		if path == "" {
			cwd, _ := os.Getwd()
			path = filepath.Join(cwd, fmt.Sprintf("gss_%d.uf2", bc.cfg.Team))
		}
		bc.out = path
	}
}

// WithGSSConfig sets up the configuration previously setup by the wizard.
func WithGSSConfig(cfg Config) BuildOption {
	return func(bc *BuildConfig) {
		bc.cfg = cfg
	}
}

// WithTeamNumber overrides the team number specified in the config
// from the wizard, and is largely only useful for doing bulk builds.
func WithTeamNumber(team int) BuildOption {
	return func(bc *BuildConfig) {
		bc.cfg.Team = team
	}
}

// WithKeepBuildDir prevents the build directory from being removed
// after the build completes.
func WithKeepBuildDir() BuildOption {
	return func(bc *BuildConfig) {
		bc.keep = true
	}
}

// WithExtractOnly sets the flag to only extract, and not to build the
// firmware image.
func WithExtractOnly() BuildOption {
	return func(bc *BuildConfig) {
		bc.extractOnly = true
	}
}
