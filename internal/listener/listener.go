// Package listener defines the network listening contract and its implementations.
package listener

import "kross/internal/conn"

// Listener implementations receive the backlog at construction time.
type Listener interface {
	Bind() error
	Listen() error
	Accept() (*conn.ClientConnection, error)
	Close() error
}
