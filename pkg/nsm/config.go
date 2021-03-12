package nsm

import (
	"errors"
	"net/url"
	"time"
)

// Config for NSM API Client
type Config struct {
	Name             string
	ConnectTo        url.URL
	DialTimeout      time.Duration
	RequestTimeout   time.Duration
	MaxTokenLifetime time.Duration
}

// IsValid checks if the configuration is valid
func (c *Config) IsValid() error {
	if c.ConnectTo.String() == "" {
		return errors.New("no NSMGr ConnectTO URL are specified")
	}
	return nil
}
