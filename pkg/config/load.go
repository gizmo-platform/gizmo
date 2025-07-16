package config

import (
	"encoding/json"
	"os"
)

// Load reads in a config from the path on disk.
func Load(path string) (*GSSConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := new(GSSConfig)
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
