// Package conn encapsule une connexion cliente acceptée.
package conn

import (
	"net"
)

// ClientConnection enveloppe une connexion TCP acceptée.
// Elle implémente io.ReadWriteCloser, ce qui permet aux fonctionnalités
// suivantes (parseur HTTP, réponses) de la lire et d'y écrire.
type ClientConnection struct {
	raw net.Conn
}

// New crée une ClientConnection à partir d'une connexion réseau brute.
func New(raw net.Conn) *ClientConnection {
	return &ClientConnection{raw: raw}
}

// Read lit des octets envoyés par le client.
func (c *ClientConnection) Read(p []byte) (int, error) {
	return c.raw.Read(p)
}

// Write envoie des octets au client.
func (c *ClientConnection) Write(p []byte) (int, error) {
	return c.raw.Write(p)
}

// Close ferme la connexion.
func (c *ClientConnection) Close() error {
	return c.raw.Close()
}

// RemoteAddr retourne l'adresse du client, par exemple "127.0.0.1:52314".
func (c *ClientConnection) RemoteAddr() string {
	return c.raw.RemoteAddr().String()
}
