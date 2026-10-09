// Package conn wraps an accepted client connection.
package conn

import (
	"net"
)

// ClientConnection wraps an accepted TCP connection.
// It implements io.ReadWriteCloser, so upcoming features
// (HTTP parser, responses) can read from it and write to it.
type ClientConnection struct {
	raw net.Conn
}

// New creates a ClientConnection from a raw network connection.
func New(raw net.Conn) *ClientConnection {
	return &ClientConnection{raw: raw}
}

// Read reads bytes sent by the client.
func (c *ClientConnection) Read(p []byte) (int, error) {
	return c.raw.Read(p)
}

// Write sends bytes to the client.
func (c *ClientConnection) Write(p []byte) (int, error) {
	return c.raw.Write(p)
}

// Close closes the connection.
func (c *ClientConnection) Close() error {
	return c.raw.Close()
}

// RemoteAddr returns the client address, for example "127.0.0.1:52314".
func (c *ClientConnection) RemoteAddr() string {
	return c.raw.RemoteAddr().String()
}
