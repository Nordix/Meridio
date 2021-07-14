package client

import (
	"errors"
	"net/url"
	"time"
)

// Config - configuration for network service client
type Config struct {
	Name             string
	RequestTimeout   time.Duration
	ConnectTo        url.URL
	MaxTokenLifetime time.Duration
}

// IsValid - check if configuration is valid
func (c *Config) IsValid() error {
	if c.Name == "" {
		return errors.New("no client name specified")
	}
	return nil
}
