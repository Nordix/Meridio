package client

import (
	"errors"
	"time"
)

// Config - configuration for network service client
type Config struct {
	Name           string        `default:"nsc" desc:"Name of Network Service Client"`
	RequestTimeout time.Duration `default:"15s" desc:"timeout to request NSE" split_words:"true"`
}

// IsValid - check if configuration is valid
func (c *Config) IsValid() error {
	if c.Name == "" {
		return errors.New("no client name specified")
	}
	return nil
}
