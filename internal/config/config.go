// Package config contient les paramètres du serveur.
package config

import (
	"fmt"
)

// ServerConfig regroupe les paramètres d'écoute.
type ServerConfig struct {
	Host    string
	Port    int
	Backlog int
}

// Default retourne une configuration utilisable sans fichier ni option.
func Default() ServerConfig {
	return ServerConfig{
		Host:    "127.0.0.1",
		Port:    8080,
		Backlog: 128,
	}
}

// Validate vérifie que les valeurs sont cohérentes.
func (c ServerConfig) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("port hors limites : %d", c.Port)
	}
	if c.Backlog < 1 {
		return fmt.Errorf("backlog doit être au moins 1 : %d", c.Backlog)
	}
	return nil
}
