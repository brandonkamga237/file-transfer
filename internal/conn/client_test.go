package conn

import (
	"io"
	"net"
	"testing"
)

func TestReadWrite(t *testing.T) {
	// net.Pipe simulates a connection without a real network.
	server, client := net.Pipe()
	defer client.Close()

	c := New(server)
	defer c.Close()

	msg := []byte("ping")

	// net.Pipe is synchronous: the write must happen in a goroutine.
	go func() {
		client.Write(msg)
	}()

	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(buf) != string(msg) {
		t.Fatalf("expected %q, got %q", msg, buf)
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
		t.Fatalf("client-side read failed: %v", err)
	}
	if string(buf) != string(msg) {
		t.Fatalf("expected %q, got %q", msg, buf)
	}
}
