// Package listener définit le contrat d'écoute réseau et ses implémentations.
package listener

import "kross/internal/conn"

// Listener est le contrat utilisé par Server.
// Le backlog est fourni à la construction de l'implémentation.
type Listener interface {
	Bind() error
	Listen() error
	Accept() (*conn.ClientConnection, error)
	Close() error
}
