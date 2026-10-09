package conn

import (
	"io"
	"net"
	"testing"
)

func TestReadWrite(t *testing.T) {
	// net.Pipe simule une connexion sans réseau réel.
	server, client := net.Pipe()
	defer client.Close()

	c := New(server)
	defer c.Close()

	msg := []byte("ping")

	// net.Pipe est synchrone : l'écriture doit se faire dans une goroutine.
	go func() {
		client.Write(msg)
	}()

	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatalf("lecture échouée : %v", err)
	}
	if string(buf) != string(msg) {
		t.Fatalf("attendu %q, reçu %q", msg, buf)
	}
}

func TestWriteToClient(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	c := New(server)
	defer c.Close()

	msg := []byte("pong")

	go func() {
		c.Write(msg)
	}()

	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatalf("lecture côté client échouée : %v", err)
	}
	if string(buf) != string(msg) {
		t.Fatalf("attendu %q, reçu %q", msg, buf)
	}
}
