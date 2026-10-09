// Package conn wraps an accepted client connection.
package conn

import (
	"net"
)

// ClientConnection implements io.ReadWriteCloser.
type ClientConnection struct {
	raw net.Conn
}

func New(raw net.Conn) *ClientConnection {
	return &ClientConnection{raw: raw}
}

func (c *ClientConnection) Read(p []byte) (int, error) {
	return c.raw.Read(p)
}

func (c *ClientConnection) Write(p []byte) (int, error) {
	return c.raw.Write(p)
}

func (c *ClientConnection) Close() error {
	return c.raw.Close()
}

func (c *ClientConnection) RemoteAddr() string {
	return c.raw.RemoteAddr().String()
}
