package config

import (
	"encoding/json"
	"github.com/pkg/errors"
)

var (
	// Version of dcos-check-runner code.
	Version = "0.4.0"
)

// Config structure is a main config object
type Config struct {
	FlagVerbose bool   `json:"verbose"`
	FlagRole    string `json:"role"`
}

// LoadFromViper takes a map of flags with values and updates the config structure.
func (c *Config) LoadFromViper(settings map[string]interface{}) error {
	// TODO(mnaboka): use a map to struct library
	body, err := json.Marshal(settings)
	if err != nil {
		return errors.Wrap(err, "unable to marshal config file")
	}

	if err := json.Unmarshal(body, c); err != nil {
		return errors.Wrap(err, "unable to unmarshal config file")
	}
	return nil
}
