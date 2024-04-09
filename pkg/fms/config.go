package fms

import (
	"os"
	"encoding/json"
)

// LoadConfig reads config off disk.
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := new(Config)
	err = json.NewDecoder(f).Decode(cfg)
	return cfg, err
}
