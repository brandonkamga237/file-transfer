// Package config holds the server settings.
package config

import (
	"fmt"
)

// ServerConfig groups the listening settings.
type ServerConfig struct {
	Host    string
	Port    int
	Backlog int
}

// Default returns a configuration usable without any file or flag.
func Default() ServerConfig {
	return ServerConfig{
		Host:    "127.0.0.1",
		Port:    8080,
		Backlog: 128,
	}
}

// Validate checks that the values are consistent.
func (c ServerConfig) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("port out of range: %d", c.Port)
	}
	if c.Backlog < 1 {
		return fmt.Errorf("backlog must be at least 1: %d", c.Backlog)
	}
	return nil
}
