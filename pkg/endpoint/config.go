package endpoint

import "time"

// Config for Endpoint
type Config struct {
	Name             string
	ServiceName      string
	Labels           map[string]string
	MaxTokenLifetime time.Duration
}
