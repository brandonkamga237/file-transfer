// Package listener defines the network listening contract and its implementations.
package listener

import "kross/internal/conn"

// Listener is the contract used by Server.
// The backlog is provided when the implementation is constructed.
type Listener interface {
	Bind() error
	Listen() error
	Accept() (*conn.ClientConnection, error)
	Close() error
}
